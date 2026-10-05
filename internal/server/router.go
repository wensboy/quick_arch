package server

import (
	"github.com/labstack/echo/v5"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

// Router 是一个前缀(通常对应一个版本)下的路由集合.
// handler 的引入被限制在 Router 实现内部, server 只负责按前缀挂载.
type Router interface {
	Prefix() string
	Mount(g *echo.Group, ctx context2.ServerContext)
}
