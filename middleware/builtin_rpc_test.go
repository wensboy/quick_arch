package middleware

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoverUnaryFactory_ConvertsPanic(t *testing.T) {
	logger := &recordingLogger{}
	interceptor := recoverUnaryFactory()(stubMiddlewareContext{log: logger})[0]

	_, err := interceptor(context.Background(), nil,
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Boom"},
		func(context.Context, any) (any, error) { panic("boom") })

	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
	if logger.msg != "panic recovered" {
		t.Fatalf("msg = %q, want panic recovered", logger.msg)
	}
}

func TestRecoverStreamFactory_ConvertsPanic(t *testing.T) {
	logger := &recordingLogger{}
	interceptor := recoverStreamFactory()(stubMiddlewareContext{log: logger})[0]

	err := interceptor(nil, nil,
		&grpc.StreamServerInfo{FullMethod: "/test.Service/Stream"},
		func(any, grpc.ServerStream) error { panic("boom") })

	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
	if logger.msg != "panic recovered" {
		t.Fatalf("msg = %q, want panic recovered", logger.msg)
	}
}
