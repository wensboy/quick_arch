package server

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v5"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

type RestServer struct {
	muxer       *echo.Echo
	server      *http.Server
	middlewares []string
	routers     []Router
}

func NewRestServer() *RestServer {
	muxer := echo.New()
	muxer.HTTPErrorHandler = handleHTTPError
	return &RestServer{
		muxer:  muxer,
		server: &http.Server{Handler: muxer},
	}
}

func (rs *RestServer) WithAddress(addr string) *RestServer {
	if addr != "" {
		rs.server.Addr = addr
	}
	return rs
}

// Use 声明按序生效的全局中间件名, 名称需已注册到 MiddlewareContext.
func (rs *RestServer) Use(names ...string) *RestServer {
	rs.middlewares = append(rs.middlewares, names...)
	return rs
}

func (rs *RestServer) Mount(routers ...Router) *RestServer {
	rs.routers = append(rs.routers, routers...)
	return rs
}

// Handler 暴露底层 http 入口, 便于测试与自定义 http.Server.
func (rs *RestServer) Handler() http.Handler { return rs.muxer }

func (rs *RestServer) Setup(ctx context2.ServerContext) {
	if ctx.MiddlewareContext != nil {
		if chain := ctx.Resolve(rs.middlewares...); len(chain) > 0 {
			rs.muxer.Use(chain...)
		}
	}
	for _, router := range rs.routers {
		router.Mount(rs.muxer.Group(router.Prefix()), ctx)
	}
}

func (rs *RestServer) Start(cancel context.CancelFunc) {
	if err := rs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
	cancel()
}

func (rs *RestServer) Stop(ctx context.Context) {
	rs.server.Shutdown(ctx)
}
