package server

import "google.golang.org/grpc"

// Service 是可挂载到 RPC 服务端的一组服务实现, 通常是 protoc 生成的 RegisterXxxServer.
// 与 Router 对称: Router 面向 rest, Service 面向 rpc.
type Service interface {
	Register(grpc.ServiceRegistrar)
}
