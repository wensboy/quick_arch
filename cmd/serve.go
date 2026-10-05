package cmd

import (
	"context"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/wensboy/quick_arch/middleware"
	"github.com/wensboy/quick_arch/router"

	context2 "github.com/wensboy/quick_arch/internal/context"
	"github.com/wensboy/quick_arch/internal/db"
	errs "github.com/wensboy/quick_arch/internal/error"
	"github.com/wensboy/quick_arch/internal/server"
)

const (
	SubCmd_Serve = "serve"

	shutdownDelay = 10 * time.Second
)

func mountServe(ctx context.Context, cmd *cli.Command) {
	cmd.Action = runServe
}

func runServe(ctx context.Context, cmd *cli.Command) error {
	srv, _, cleanup, err := setupServe(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	runner := server.NewServerRunner()
	runner.SetServer(srv)
	runner.Run(func() {}, shutdownDelay)
	return nil
}

// setupServe 装配服务上下文与服务实例, 但不启动监听.
func setupServe(ctx context.Context) (server.Server, *context2.ServerContext, func(), error) {
	app, ok := context2.AppFrom(ctx)
	if !ok {
		return nil, nil, nil, errs.New(server.ErrAppContext)
	}

	serverCfg := server.Load(app.ConfigContext)

	dbManager, err := db.Open(ctx, app.ConfigContext)
	if err != nil {
		return nil, nil, nil, err
	}
	cleanup := func() { _ = dbManager.Close() }

	// 中间件上下文需要访问日志/配置/数据库, 由组合根统一注入.
	mw := server.NewMiddlewareStore().Bind(app.LogContext, app.ConfigContext, dbManager)
	middleware.RegisterMiddlewares(mw, app.ConfigContext)

	sctx := context2.NewServerContext(app.ConfigContext, app.LogContext, dbManager, mw, app.EmbedContext)

	srv, err := server.New(serverCfg, router.Routers())
	if err != nil {
		cleanup()
		return nil, nil, nil, err
	}
	srv.Setup(*sctx)

	if app.LogContext != nil {
		app.Log("info", "serve ready",
			"kind", serverCfg.Kind,
			"address", serverCfg.Address,
			"middleware", serverCfg.Middlewares,
			"db", dbManager.Names(),
		)
	}
	return srv, sctx, cleanup, nil
}
