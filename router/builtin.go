package router

import (
	"github.com/labstack/echo/v5"

	"github.com/wensboy/quick_arch/handler"
	context2 "github.com/wensboy/quick_arch/internal/context"
)

// mountBuiltin 注册 builtin 领域的全部端点: 不需要数据库操作, 由 handler 直接返回.
func mountBuiltin(g *echo.Group, ctx context2.ServerContext) {
	builtin := handler.NewBuiltinHandler(ctx.ConfigContext, ctx.EmbedContext)
	g.GET("/ping", builtin.Ping)
	g.GET("/scalar", builtin.Scalar)
	g.GET("/openapi.json", builtin.OpenAPI)
}
