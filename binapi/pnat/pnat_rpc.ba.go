// Code generated by GoVPP's binapi-generator. DO NOT EDIT.

package pnat

import (
	"context"
	"fmt"
	"io"

	api "go.fd.io/govpp/api"
)

// RPCService defines RPC service pnat.
type RPCService interface {
	PnatBindingAdd(ctx context.Context, in *PnatBindingAdd) (*PnatBindingAddReply, error)
	PnatBindingAddV2(ctx context.Context, in *PnatBindingAddV2) (*PnatBindingAddV2Reply, error)
	PnatBindingAttach(ctx context.Context, in *PnatBindingAttach) (*PnatBindingAttachReply, error)
	PnatBindingDel(ctx context.Context, in *PnatBindingDel) (*PnatBindingDelReply, error)
	PnatBindingDetach(ctx context.Context, in *PnatBindingDetach) (*PnatBindingDetachReply, error)
	PnatBindingsGet(ctx context.Context, in *PnatBindingsGet) (RPCService_PnatBindingsGetClient, error)
	PnatInterfacesGet(ctx context.Context, in *PnatInterfacesGet) (RPCService_PnatInterfacesGetClient, error)
}

type serviceClient struct {
	conn api.Connection
}

func NewServiceClient(conn api.Connection) RPCService {
	return &serviceClient{conn}
}

func (c *serviceClient) PnatBindingAdd(ctx context.Context, in *PnatBindingAdd) (*PnatBindingAddReply, error) {
	out := new(PnatBindingAddReply)
	err := c.conn.Invoke(ctx, in, out)
	if err != nil {
		return nil, err
	}
	return out, api.RetvalToVPPApiError(out.Retval)
}

func (c *serviceClient) PnatBindingAddV2(ctx context.Context, in *PnatBindingAddV2) (*PnatBindingAddV2Reply, error) {
	out := new(PnatBindingAddV2Reply)
	err := c.conn.Invoke(ctx, in, out)
	if err != nil {
		return nil, err
	}
	return out, api.RetvalToVPPApiError(out.Retval)
}

func (c *serviceClient) PnatBindingAttach(ctx context.Context, in *PnatBindingAttach) (*PnatBindingAttachReply, error) {
	out := new(PnatBindingAttachReply)
	err := c.conn.Invoke(ctx, in, out)
	if err != nil {
		return nil, err
	}
	return out, api.RetvalToVPPApiError(out.Retval)
}

func (c *serviceClient) PnatBindingDel(ctx context.Context, in *PnatBindingDel) (*PnatBindingDelReply, error) {
	out := new(PnatBindingDelReply)
	err := c.conn.Invoke(ctx, in, out)
	if err != nil {
		return nil, err
	}
	return out, api.RetvalToVPPApiError(out.Retval)
}

func (c *serviceClient) PnatBindingDetach(ctx context.Context, in *PnatBindingDetach) (*PnatBindingDetachReply, error) {
	out := new(PnatBindingDetachReply)
	err := c.conn.Invoke(ctx, in, out)
	if err != nil {
		return nil, err
	}
	return out, api.RetvalToVPPApiError(out.Retval)
}

func (c *serviceClient) PnatBindingsGet(ctx context.Context, in *PnatBindingsGet) (RPCService_PnatBindingsGetClient, error) {
	stream, err := c.conn.NewStream(ctx)
	if err != nil {
		return nil, err
	}
	x := &serviceClient_PnatBindingsGetClient{stream}
	if err := x.Stream.SendMsg(in); err != nil {
		return nil, err
	}
	return x, nil
}

type RPCService_PnatBindingsGetClient interface {
	Recv() (*PnatBindingsDetails, error)
	api.Stream
}

type serviceClient_PnatBindingsGetClient struct {
	api.Stream
}

func (c *serviceClient_PnatBindingsGetClient) Recv() (*PnatBindingsDetails, error) {
	msg, err := c.Stream.RecvMsg()
	if err != nil {
		return nil, err
	}
	switch m := msg.(type) {
	case *PnatBindingsDetails:
		return m, nil
	case *PnatBindingsGetReply:
		err = c.Stream.Close()
		if err != nil {
			return nil, err
		}
		return nil, io.EOF
	default:
		return nil, fmt.Errorf("unexpected message: %T %v", m, m)
	}
}

func (c *serviceClient) PnatInterfacesGet(ctx context.Context, in *PnatInterfacesGet) (RPCService_PnatInterfacesGetClient, error) {
	stream, err := c.conn.NewStream(ctx)
	if err != nil {
		return nil, err
	}
	x := &serviceClient_PnatInterfacesGetClient{stream}
	if err := x.Stream.SendMsg(in); err != nil {
		return nil, err
	}
	return x, nil
}

type RPCService_PnatInterfacesGetClient interface {
	Recv() (*PnatInterfacesDetails, error)
	api.Stream
}

type serviceClient_PnatInterfacesGetClient struct {
	api.Stream
}

func (c *serviceClient_PnatInterfacesGetClient) Recv() (*PnatInterfacesDetails, error) {
	msg, err := c.Stream.RecvMsg()
	if err != nil {
		return nil, err
	}
	switch m := msg.(type) {
	case *PnatInterfacesDetails:
		return m, nil
	case *PnatInterfacesGetReply:
		err = c.Stream.Close()
		if err != nil {
			return nil, err
		}
		return nil, io.EOF
	default:
		return nil, fmt.Errorf("unexpected message: %T %v", m, m)
	}
}
