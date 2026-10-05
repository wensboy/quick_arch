package router

import "github.com/wensboy/quick_arch/internal/server"

// Routers 返回全部路由集合, 由组合根挂载到 server.
func Routers() []server.Router {
	return []server.Router{v1{}}
}
