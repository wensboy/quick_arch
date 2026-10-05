package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/wensboy/quick_arch/handler"
	"github.com/wensboy/quick_arch/middleware"

	"github.com/wensboy/quick_arch/internal/config"
	context2 "github.com/wensboy/quick_arch/internal/context"
	errs "github.com/wensboy/quick_arch/internal/error"
	"github.com/wensboy/quick_arch/internal/log"
	"github.com/wensboy/quick_arch/internal/server"

	cli2 "github.com/wensboy/quick_arch/internal/cli"
)

// 嵌入 fs 会保留 data/conf 前缀.
const (
	commandConfigPath = "data/conf/command.json"
	appConfigPath     = "data/conf/config.json"
)

func Execute(embedContext context2.EmbedContext) int {
	confFS, ok := embedContext.GetFS("conf")
	if !ok {
		fmt.Fprintln(os.Stderr, `quick_arch: embedded fs "conf" not found`)
		return 1
	}
	commandStr, err := confFS.ReadFile(commandConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "quick_arch: read command source: %v\n", err)
		return 1
	}
	rootCmd, err := cli2.ParseString(commandStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "quick_arch: parse command config: %v\n", err)
		return 1
	}

	registry := newRegistry()
	registerFlags(registry, rootCmd)

	// flag 取值需 Run 解析后才可见, 故在 Before 中构建 logger 与 AppContext.
	var appLogger *log.Logger
	rootCmd.Before = func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
		cfgStore, err := buildConfigStore(embedContext, registry, cmd)
		if err != nil {
			return ctx, err
		}
		appLogger, err = log.New(cfgStore)
		if err != nil {
			return ctx, errs.Wrap(errs.ErrInternal, err).With("stage", "logger")
		}
		return context2.AppContext{
			Context:       ctx,
			EmbedContext:  embedContext,
			ConfigContext: cfgStore,
			LogContext:    appLogger,
		}, nil
	}
	rootCmd.After = func(ctx context.Context, cmd *cli.Command) error {
		if appLogger != nil {
			_ = appLogger.Sync()
		}
		return nil
	}

	mountCmd(context.Background(), rootCmd, SubCmd_Version, mountVersion)
	mountCmd(context.Background(), rootCmd, SubCmd_Generate, mountGenerate)
	mountCmd(context.Background(), rootCmd, SubCmd_Serve, mountServe)

	if err = rootCmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "quick_arch: %v\n", err)
		return 1
	}
	return 0
}

func newRegistry() *config.Registry {
	registry := config.NewRegistry()
	log.RegisterConfig(registry)
	server.RegisterConfig(registry)
	middleware.RegisterConfig(registry)
	handler.RegisterConfig(registry)
	return registry
}

func registerFlags(registry *config.Registry, cmd *cli.Command) {
	for _, entry := range registry.Entries() {
		if entry.Flag == "" {
			continue
		}
		cmd.Flags = append(cmd.Flags, cli2.BuildFlag(cli2.Flag{
			Name:        entry.Flag,
			Aliases:     entry.Aliases,
			Usage:       entry.Usage,
			Type:        entry.Type,
			DefaultText: flagDefaultText(entry),
		}))
	}
}

func flagDefaultText(entry config.Entry) string {
	if entry.Default == nil || entry.Type == config.FlagTypeBool {
		return ""
	}
	return fmt.Sprint(entry.Default)
}

func buildConfigStore(embedContext context2.EmbedContext, registry *config.Registry, cmd *cli.Command) (context2.ConfigContext, error) {
	var fileData map[string]any
	if confFS, ok := embedContext.GetFS("conf"); ok {
		raw, err := config.ReadJSON(confFS, appConfigPath)
		if err != nil {
			return nil, err
		}
		fileData = raw
	}

	return config.Build(config.Options{
		File:     fileData,
		Registry: registry,
		Flags:    cli2.FlagToSource(cmd),
	}), nil
}

func mountCmd(ctx context.Context, rootCmd *cli.Command, subName string, fn func(context.Context, *cli.Command)) {
	for _, cmd := range rootCmd.Commands {
		if cmd.Name == subName {
			fn(ctx, cmd)
			return
		}
	}
	fmt.Fprintf(os.Stderr, "quick_arch: sub command %q not found\n", subName)
}
