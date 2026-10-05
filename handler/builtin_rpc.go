package handler

import (
	"context"

	"google.golang.org/grpc"

	builtinpb "github.com/wensboy/quick_arch/proto/builtin"
)

func NewBuiltinRPC() *BuiltinRPC {
	return &BuiltinRPC{}
}

// Register 让 BuiltinRPC 满足 server.Service, 由 rpc 服务端装配时调用.
func (h *BuiltinRPC) Register(r grpc.ServiceRegistrar) {
	builtinpb.RegisterBuiltinServer(r, h)
}

func (h *BuiltinRPC) Ping(context.Context, *builtinpb.PingRequest) (*builtinpb.PingResponse, error) {
	return &builtinpb.PingResponse{Message: "pong"}, nil
}

// BuiltinRPC 是 builtin 领域的 gRPC 实现, 与 REST 侧的 /api/v1/ping 同域同义.
type BuiltinRPC struct {
	builtinpb.UnimplementedBuiltinServer
}

var _ builtinpb.BuiltinServer = (*BuiltinRPC)(nil)
