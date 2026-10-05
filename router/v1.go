package router

import (
	"github.com/labstack/echo/v5"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

func (v1) Prefix() string { return "/api/v1" }

// Mount 按领域装配本版本的路由; 需要数据库操作的领域才走 handler -> service -> repo -> db.
func (v1) Mount(g *echo.Group, ctx context2.ServerContext) {
	mountBuiltin(g, ctx)
}

type v1 struct{}
