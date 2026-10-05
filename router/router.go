package router

import (
	"github.com/wensboy/quick_arch/handler"
	"github.com/wensboy/quick_arch/internal/server"
)

// Routers 返回全部 rest 路由集合, 由组合根挂载到 rest 服务端.
func Routers() []server.Router {
	return []server.Router{v1{}}
}

// Services 返回全部 rpc 服务实现, 由组合根挂载到 rpc 服务端.
func Services() []server.Service {
	return []server.Service{
		handler.NewBuiltinRPC(),
	}
}
