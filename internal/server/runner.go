package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	context2 "github.com/wensboy/quick_arch/internal/context"
)

type Server interface {
	Setup(context2.ServerContext)
	Start(context.CancelFunc)
	Stop(context.Context)
}

type ServerRunner struct {
	Server
	quit chan os.Signal
}

func NewServerRunner() *ServerRunner {
	return &ServerRunner{
		quit: make(chan os.Signal, 1),
	}
}

func (sr *ServerRunner) SetServer(s Server) {
	sr.Server = s
}

func (sr *ServerRunner) Run(gc func(), delay time.Duration) int {
	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		sr.Server.Start(cancel)
	}()
	signal.Notify(sr.quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGABRT)
	select {
	case <-ctx.Done():
		fmt.Fprintf(os.Stderr, "server exit...\n")
	case <-sr.quit:
		fmt.Fprintf(os.Stderr, "signal exit...\n")
		ctx, cancel = context.WithTimeout(context.TODO(), delay)
		defer cancel()
		sr.Server.Stop(ctx)
	}
	gc()
	return 0
}

func (sr *ServerRunner) Exit() {
	sr.quit <- syscall.SIGQUIT
}
