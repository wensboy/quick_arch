package handler

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	builtinpb "github.com/wensboy/quick_arch/proto/builtin"
)

func TestBuiltinRPC_Ping(t *testing.T) {
	resp, err := NewBuiltinRPC().Ping(context.Background(), &builtinpb.PingRequest{})
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if resp.GetMessage() != "pong" {
		t.Fatalf("message = %q, want pong", resp.GetMessage())
	}
}

func TestBuiltinRPC_Register(t *testing.T) {
	registrar := grpc.NewServer()
	NewBuiltinRPC().Register(registrar)

	name := builtinpb.Builtin_ServiceDesc.ServiceName
	if _, ok := registrar.GetServiceInfo()[name]; !ok {
		t.Fatalf("service %q not registered: %v", name, registrar.GetServiceInfo())
	}
}
