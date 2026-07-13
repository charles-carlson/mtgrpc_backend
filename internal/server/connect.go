package server

import (
	"context"
	"errors"

	"backend_nonsense/pb"
	"backend_nonsense/pb/pbconnect"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ConnectAdapter exposes the existing grpc-style *Server through the Connect
// MTGRPCHandler interface, so the same service logic is served over the
// Connect, gRPC, and gRPC-Web protocols (browser-reachable) from one handler.
type ConnectAdapter struct {
	impl *Server
}

// compile-time check that we satisfy the generated handler interface.
var _ pbconnect.MTGRPCHandler = (*ConnectAdapter)(nil)

func NewConnectAdapter(impl *Server) *ConnectAdapter {
	return &ConnectAdapter{impl: impl}
}

// toConnectErr preserves the gRPC status code set by the service layer.
// gRPC and Connect share the same numeric code space, so a non-OK gRPC code
// maps directly onto the matching Connect code.
func toConnectErr(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() == codes.OK {
		return connect.NewError(connect.CodeUnknown, err)
	}
	return connect.NewError(connect.Code(st.Code()), errors.New(st.Message()))
}

func (a *ConnectAdapter) AddCard(ctx context.Context, req *connect.Request[pb.AddCardRequest]) (*connect.Response[pb.AddCardResponse], error) {
	resp, err := a.impl.AddCard(ctx, req.Msg)
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(resp), nil
}

func (a *ConnectAdapter) GetCard(ctx context.Context, req *connect.Request[pb.GetCardRequest]) (*connect.Response[pb.GetCardResponse], error) {
	resp, err := a.impl.GetCard(ctx, req.Msg)
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(resp), nil
}

func (a *ConnectAdapter) GetCardsByName(ctx context.Context, req *connect.Request[pb.GetCardsByNameRequest]) (*connect.Response[pb.GetCardsByNameResponse], error) {
	resp, err := a.impl.GetCardsByName(ctx, req.Msg)
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(resp), nil
}

func (a *ConnectAdapter) GetCardsBySet(ctx context.Context, req *connect.Request[pb.GetCardsBySetRequest]) (*connect.Response[pb.GetCardsBySetResponse], error) {
	resp, err := a.impl.GetCardsBySet(ctx, req.Msg)
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(resp), nil
}

func (a *ConnectAdapter) SearchCards(ctx context.Context, req *connect.Request[pb.SearchCardsRequest]) (*connect.Response[pb.SearchCardsResponse], error) {
	resp, err := a.impl.SearchCards(ctx, req.Msg)
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(resp), nil
}

func (a *ConnectAdapter) ListCards(ctx context.Context, req *connect.Request[pb.ListCardsRequest]) (*connect.Response[pb.ListCardsResponse], error) {
	resp, err := a.impl.ListCards(ctx, req.Msg)
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(resp), nil
}

func (a *ConnectAdapter) ListSets(ctx context.Context, req *connect.Request[pb.ListSetsRequest]) (*connect.Response[pb.ListSetsResponse], error) {
	resp, err := a.impl.ListSets(ctx, req.Msg)
	if err != nil {
		return nil, toConnectErr(err)
	}
	return connect.NewResponse(resp), nil
}
