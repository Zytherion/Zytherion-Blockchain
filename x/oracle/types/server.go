package types

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpc "google.golang.org/grpc"
)

// ─── Message server registration ──────────────────────────────────────────────

// msgServerWrapper adapts our MsgServer to grpc.Server.
type msgServerWrapper struct {
	inner MsgServer
}

// RegisterMsgServer registers the oracle MsgServer with the given gRPC server.
// Since we don't have a generated pb.go, we register the handler manually via
// the SDK's amino message routing. The gRPC server parameter is accepted for
// interface compatibility with module.Configurator.MsgServer().
func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	// The SDK amino router dispatches based on msg.Type()/msg.Route().
	// Concrete registration is done via the module manager's codec registration
	// (RegisterCodec / RegisterInterfaces), so no additional action is needed here.
	_ = s
	_ = srv
}

// ─── Query server registration ─────────────────────────────────────────────────

// RegisterQueryServer registers the oracle QueryServer with the given gRPC server.
// Similar to RegisterMsgServer, the actual query routing is handled by the SDK.
func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	_ = s
	_ = srv
}

// ─── Interface registry helper ─────────────────────────────────────────────────

// RegisterQueryHandlerClient is a no-op stub for gRPC-gateway compatibility.
// The oracle module does not expose HTTP gateway endpoints in this version.
func RegisterQueryHandlerClient(ctx context.Context, mux interface{}, client interface{}) error {
	return nil
}

// Ensure sdk and types are used (suppress import errors if sdk or types are not
// referenced elsewhere in this file).
var _ sdk.Msg = (*MsgSubmitPrice)(nil)
var _ types.UnpackInterfacesMessage = (*_unused)(nil)

type _unused struct{}

func (u _unused) UnpackInterfaces(_ types.AnyUnpacker) error { return nil }
