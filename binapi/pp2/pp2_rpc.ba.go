// Code generated by GoVPP's binapi-generator. DO NOT EDIT.

package pp2

import (
	"context"

	api "go.fd.io/govpp/api"
)

// RPCService defines RPC service pp2.
type RPCService interface {
	MrvlPp2Create(ctx context.Context, in *MrvlPp2Create) (*MrvlPp2CreateReply, error)
	MrvlPp2Delete(ctx context.Context, in *MrvlPp2Delete) (*MrvlPp2DeleteReply, error)
}

type serviceClient struct {
	conn api.Connection
}

func NewServiceClient(conn api.Connection) RPCService {
	return &serviceClient{conn}
}

func (c *serviceClient) MrvlPp2Create(ctx context.Context, in *MrvlPp2Create) (*MrvlPp2CreateReply, error) {
	out := new(MrvlPp2CreateReply)
	err := c.conn.Invoke(ctx, in, out)
	if err != nil {
		return nil, err
	}
	return out, api.RetvalToVPPApiError(out.Retval)
}

func (c *serviceClient) MrvlPp2Delete(ctx context.Context, in *MrvlPp2Delete) (*MrvlPp2DeleteReply, error) {
	out := new(MrvlPp2DeleteReply)
	err := c.conn.Invoke(ctx, in, out)
	if err != nil {
		return nil, err
	}
	return out, api.RetvalToVPPApiError(out.Retval)
}
