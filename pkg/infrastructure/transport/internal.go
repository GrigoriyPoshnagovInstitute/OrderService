package transport

import (
	"context"

	api "github.com/GrigoriyPoshnagovInstitute/OrderService/api/server/orderinternal"
)

func NewInternalAPI() api.OrderInternalServiceServer {
	return &internalAPI{}
}

type internalAPI struct {
	api.UnimplementedOrderInternalServiceServer
}

func (i *internalAPI) Ping(_ context.Context, _ *api.PingRequest) (*api.PingResponse, error) {
	return &api.PingResponse{
		Message: "pong",
	}, nil
}
