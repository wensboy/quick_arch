package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

var (
	APP_NAME          = "unknown"
	APP_BUILD_TIME    = "-"
	APP_BUILD_VERSION = "unknown"
)

const (
	SubCmd_Version = "version"
)

func mountVersion(ctx context.Context, cmd *cli.Command) {
	cmd.Action = func(_ context.Context, c *cli.Command) error {
		out := c.Writer
		if out == nil {
			out = os.Stdout
		}
		if c.Bool("short") {
			fmt.Fprintln(out, APP_BUILD_VERSION)
			return nil
		}
		fmt.Fprintf(out, "%s\n  version:    %s\n  build time: %s\n",
			APP_NAME, APP_BUILD_VERSION, APP_BUILD_TIME)
		return nil
	}
}
