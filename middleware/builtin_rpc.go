package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

// recoverUnaryFactory 是 recover 的 gRPC 一元形态: panic 转为 codes.Internal 状态.
func recoverUnaryFactory() context2.UnaryInterceptorFactory {
	return func(mc context2.MiddlewareContext) []grpc.UnaryServerInterceptor {
		return []grpc.UnaryServerInterceptor{func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
			defer func() {
				if r := recover(); r != nil {
					logAt(mc, "error", "panic recovered", "method", info.FullMethod, "panic", r)
					err = status.Error(codes.Internal, "internal error")
				}
			}()
			return handler(ctx, req)
		}}
	}
}

// recoverStreamFactory 是 recover 的 gRPC 流形态: panic 转为 codes.Internal 状态.
func recoverStreamFactory() context2.StreamInterceptorFactory {
	return func(mc context2.MiddlewareContext) []grpc.StreamServerInterceptor {
		return []grpc.StreamServerInterceptor{func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logAt(mc, "error", "panic recovered", "method", info.FullMethod, "panic", r)
					err = status.Error(codes.Internal, "internal error")
				}
			}()
			return handler(srv, ss)
		}}
	}
}
