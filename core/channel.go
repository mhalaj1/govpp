// Copyright (c) 2018 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"go.fd.io/govpp/adapter"
	"go.fd.io/govpp/api"
)

var (
	ErrInvalidRequestCtx = errors.New("invalid request context")
)

// MessageCodec provides functionality for decoding binary data to generated API messages.
type MessageCodec interface {
	// EncodeMsg encodes message into binary data.
	EncodeMsg(msg api.Message, msgID uint16) ([]byte, error)
	// DecodeMsg decodes binary-encoded data of a message into provided Message structure.
	DecodeMsg(data []byte, msg api.Message) error
	// DecodeMsgContext decodes context from message data and type.
	DecodeMsgContext(data []byte, msgType api.MessageType) (context uint32, err error)
}

// MessageIdentifier provides identification of generated API messages.
type MessageIdentifier interface {
	// GetMessageID returns message identifier of given API message.
	GetMessageID(msg api.Message) (uint16, error)
	// GetMessagePath returns path for the given message
	GetMessagePath(msg api.Message) string
	// LookupByID looks up message name and crc by ID
	LookupByID(path string, msgID uint16) (api.Message, error)
}

// vppRequest is a request that will be sent to VPP.
type vppRequest struct {
	seqNum uint16      // sequence number
	msg    api.Message // binary API message to be send to VPP
	multi  bool        // true if multipart response is expected
}

// vppReply is a reply received from VPP.
type vppReply struct {
	seqNum       uint16 // sequence number
	msgID        uint16 // ID of the message
	data         []byte // encoded data with the message
	lastReceived bool   // for multi request, true if the last reply has been already received
	err          error  // in case of error, data is nil and this member contains error
}

// requestCtx is a context for request with single reply
type requestCtx struct {
	ch     *Channel
	seqNum uint16
}

// multiRequestCtx is a context for request with multiple responses
type multiRequestCtx struct {
	ch     *Channel
	seqNum uint16
}

// subscriptionCtx is a context of subscription for delivery of specific notification messages.
type subscriptionCtx struct {
	ch         *Channel
	notifChan  chan api.Message   // channel where notification messages will be delivered to
	msgID      uint16             // message ID for the subscribed event message
	event      api.Message        // event message that this subscription is for
	msgFactory func() api.Message // function that returns a new instance of the specific message that is expected as a notification
}

// Channel is the main communication interface with govpp core. It contains four Go channels, one for sending the requests
// to VPP, one for receiving the replies from it and the same set for notifications. The user can access the Go channels
// via methods provided by Channel interface in this package. Do not use the same channel from multiple goroutines
// concurrently, otherwise the responses could mix! Use multiple channels instead.
type Channel struct {
	id   uint16
	conn *Connection

	reqChan   chan *vppRequest // channel for sending the requests to VPP
	replyChan chan *vppReply   // channel where VPP replies are delivered to

	msgCodec      MessageCodec      // used to decode binary data to generated API messages
	msgIdentifier MessageIdentifier // used to retrieve message ID of a message

	lastSeqNum uint16 // sequence number of the last sent request

	delayedReply        *vppReply     // reply already taken from ReplyChan, buffered for later delivery
	replyTimeout        time.Duration // maximum time that the API waits for a reply from VPP before returning an error, can be set with SetReplyTimeout
	receiveReplyTimeout time.Duration // maximum time that we wait for receiver to consume reply
}

func (c *Connection) newChannel(reqChanBufSize, replyChanBufSize int) (*Channel, error) {
	// get a channel from the pool
	channel := c.channelPool.Get()
	if channel == nil {
		return nil, errors.New("all channel IDs are in use")
	}
	if cap(channel.reqChan) != reqChanBufSize {
		channel.reqChan = make(chan *vppRequest, reqChanBufSize)
	}
	if cap(channel.replyChan) != replyChanBufSize {
		channel.replyChan = make(chan *vppReply, replyChanBufSize)
	}

	// store API channel within the client
	c.channelsLock.Lock()
	c.channels[channel.id] = channel
	c.channelsLock.Unlock()
	return channel, nil
}

func (ch *Channel) GetID() uint16 {
	return ch.id
}

func (ch *Channel) SendRequest(msg api.Message) api.RequestCtx {
	req := ch.newRequest(msg, false)
	ch.reqChan <- req
	return &requestCtx{ch: ch, seqNum: req.seqNum}
}

func (ch *Channel) SendMultiRequest(msg api.Message) api.MultiRequestCtx {
	req := ch.newRequest(msg, true)
	ch.reqChan <- req
	return &multiRequestCtx{ch: ch, seqNum: req.seqNum}
}

func (ch *Channel) nextSeqNum() uint16 {
	ch.lastSeqNum++
	return ch.lastSeqNum
}

func (ch *Channel) newRequest(msg api.Message, multi bool) *vppRequest {
	return &vppRequest{
		msg:    msg,
		seqNum: ch.nextSeqNum(),
		multi:  multi,
	}
}

func (ch *Channel) CheckCompatiblity(msgs ...api.Message) error {
	var comperr api.CompatibilityError
	for _, msg := range msgs {
		_, err := ch.msgIdentifier.GetMessageID(msg)
		if err != nil {
			if uerr, ok := err.(*adapter.UnknownMsgError); ok {
				comperr.IncompatibleMessages = append(comperr.IncompatibleMessages, getMsgID(uerr.MsgName, uerr.MsgCrc))
				continue
			}
			// other errors return immediatelly
			return err
		}
		comperr.CompatibleMessages = append(comperr.CompatibleMessages, getMsgNameWithCrc(msg))
	}
	if len(comperr.IncompatibleMessages) == 0 {
		return nil
	}
	return &comperr
}

func (ch *Channel) SubscribeNotification(notifChan chan api.Message, event api.Message) (api.SubscriptionCtx, error) {
	msgID, err := ch.msgIdentifier.GetMessageID(event)
	if err != nil {
		log.WithFields(logrus.Fields{
			"msg_name": event.GetMessageName(),
			"msg_crc":  event.GetCrcString(),
		}).Errorf("unable to retrieve message ID: %v", err)
		return nil, fmt.Errorf("unable to retrieve event message ID: %v", err)
	}

	sub := &subscriptionCtx{
		ch:         ch,
		notifChan:  notifChan,
		msgID:      msgID,
		event:      event,
		msgFactory: getMsgFactory(event),
	}

	// add the subscription into map
	ch.conn.subscriptionsLock.Lock()
	defer ch.conn.subscriptionsLock.Unlock()

	ch.conn.subscriptions[msgID] = append(ch.conn.subscriptions[msgID], sub)

	return sub, nil
}

func (ch *Channel) SetReplyTimeout(timeout time.Duration) {
	ch.replyTimeout = timeout
}

func (ch *Channel) Close() {
	close(ch.reqChan)
}

func (req *requestCtx) ReceiveReply(msg api.Message) error {
	if req == nil || req.ch == nil {
		return ErrInvalidRequestCtx
	}

	lastReplyReceived, err := req.ch.receiveReplyInternal(msg, req.seqNum)
	if err != nil {
		return err
	} else if lastReplyReceived {
		return errors.New("multipart reply recieved while a single reply expected")
	}

	return nil
}

func (req *multiRequestCtx) ReceiveReply(msg api.Message) (lastReplyReceived bool, err error) {
	if req == nil || req.ch == nil {
		return false, ErrInvalidRequestCtx
	}

	return req.ch.receiveReplyInternal(msg, req.seqNum)
}

func (sub *subscriptionCtx) Unsubscribe() error {
	log.WithFields(logrus.Fields{
		"msg_name": sub.event.GetMessageName(),
		"msg_id":   sub.msgID,
	}).Debug("Removing notification subscription.")

	// remove the subscription from the map
	sub.ch.conn.subscriptionsLock.Lock()
	defer sub.ch.conn.subscriptionsLock.Unlock()

	for i, item := range sub.ch.conn.subscriptions[sub.msgID] {
		if item == sub {
			// close notification channel
			close(sub.ch.conn.subscriptions[sub.msgID][i].notifChan)
			// remove i-th item in the slice
			sub.ch.conn.subscriptions[sub.msgID] = append(sub.ch.conn.subscriptions[sub.msgID][:i], sub.ch.conn.subscriptions[sub.msgID][i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("subscription for %q not found", sub.event.GetMessageName())
}

const maxInt64 = 1<<63 - 1

// receiveReplyInternal receives a reply from the reply channel into the provided msg structure.
func (ch *Channel) receiveReplyInternal(msg api.Message, expSeqNum uint16) (lastReplyReceived bool, err error) {
	if msg == nil {
		return false, errors.New("nil message passed in")
	}

	var ignore bool

	if vppReply := ch.delayedReply; vppReply != nil {
		// try the delayed reply
		ch.delayedReply = nil
		ignore, lastReplyReceived, err = ch.processReply(vppReply, expSeqNum, msg)
		if !ignore {
			return lastReplyReceived, err
		}
	}

	slowReplyDur := WarnSlowReplyDuration
	timeout := ch.replyTimeout
	if timeout <= 0 {
		timeout = maxInt64
	}
	timeoutTimer := time.NewTimer(timeout)
	slowTimer := time.NewTimer(slowReplyDur)
	for {
		select {
		// blocks until a reply comes to ReplyChan or until timeout expires
		case vppReply := <-ch.replyChan:
			ignore, lastReplyReceived, err = ch.processReply(vppReply, expSeqNum, msg)
			if ignore {
				log.WithFields(logrus.Fields{
					"expSeqNum": expSeqNum,
					"channel":   ch.id,
				}).Warnf("ignoring received reply: %+v (expecting: %s)", vppReply, msg.GetMessageName())
				continue
			}
			return lastReplyReceived, err
		case <-slowTimer.C:
			log.WithFields(logrus.Fields{
				"expSeqNum": expSeqNum,
				"channel":   ch.id,
			}).Warnf("reply is taking too long (>%v): %v ", slowReplyDur, msg.GetMessageName())
			continue
		case <-timeoutTimer.C:
			log.WithFields(logrus.Fields{
				"expSeqNum": expSeqNum,
				"channel":   ch.id,
			}).Debugf("timeout (%v) waiting for reply: %s", timeout, msg.GetMessageName())
			err = fmt.Errorf("no reply received within the timeout period %s", timeout)
			return false, err
		}
	}
}

func (ch *Channel) processReply(reply *vppReply, expSeqNum uint16, msg api.Message) (ignore bool, lastReplyReceived bool, err error) {
	// check the sequence number
	cmpSeqNums := compareSeqNumbers(reply.seqNum, expSeqNum)
	if cmpSeqNums == -1 {
		// reply received too late, ignore the message
		log.WithField("seqNum", reply.seqNum).
			Warn("Received reply to an already closed binary API request")
		ignore = true
		return
	}
	if cmpSeqNums == 1 {
		ch.delayedReply = reply
		err = fmt.Errorf("missing binary API reply with sequence number: %d", expSeqNum)
		return
	}

	if reply.err != nil {
		err = reply.err
		return
	}
	if reply.lastReceived {
		lastReplyReceived = true
		return
	}

	// message checks
	var expMsgID uint16
	expMsgID, err = ch.msgIdentifier.GetMessageID(msg)
	if err != nil {
		err = fmt.Errorf("message %s with CRC %s is not compatible with the VPP we are connected to",
			msg.GetMessageName(), msg.GetCrcString())
		return
	}

	if reply.msgID != expMsgID {
		var msgNameCrc string
		pkgPath := ch.msgIdentifier.GetMessagePath(msg)
		if replyMsg, err := ch.msgIdentifier.LookupByID(pkgPath, reply.msgID); err != nil {
			msgNameCrc = err.Error()
		} else {
			msgNameCrc = getMsgNameWithCrc(replyMsg)
		}

		err = fmt.Errorf("received unexpected message (seqNum=%d), expected %s (ID %d), but got %s (ID %d) "+
			"(check if multiple goroutines are not sharing single GoVPP channel)",
			reply.seqNum, msg.GetMessageName(), expMsgID, msgNameCrc, reply.msgID)
		return
	}

	// decode the message
	if err = ch.msgCodec.DecodeMsg(reply.data, msg); err != nil {
		return
	}

	// check Retval and convert it into VnetAPIError error
	if strings.HasSuffix(msg.GetMessageName(), "_reply") {
		// TODO: use categories for messages to avoid checking message name
		if f := reflect.Indirect(reflect.ValueOf(msg)).FieldByName("Retval"); f.IsValid() {
			var retval int32
			switch f.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				retval = int32(f.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				retval = int32(f.Uint())
			default:
				logrus.Warnf("invalid kind (%v) for Retval field of message %v", f.Kind(), msg.GetMessageName())
			}
			err = api.RetvalToVPPApiError(retval)
		}
	}

	return
}

func (ch *Channel) Reset() {
	if len(ch.reqChan) > 0 || len(ch.replyChan) > 0 {
		log.WithField("channel", ch.id).Debugf("draining channel buffers (req: %d, reply: %d)", len(ch.reqChan), len(ch.replyChan))
	}
	// Drain any lingering items in the buffers
	for empty := false; !empty; {
		// channels must be set to nil when closed to prevent
		// select below to always run the case immediatelly
		// which would make the loop run forever
		select {
		case _, ok := <-ch.reqChan:
			if !ok {
				ch.reqChan = nil
			}
		case _, ok := <-ch.replyChan:
			if !ok {
				ch.replyChan = nil
			}
		default:
			empty = true
		}
	}
}
