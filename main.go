//go:build !js

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/thomaslaurenson/cellmate/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := cmd.NewRootCmd(os.Stdout, os.Stderr, cmd.RunGame, cmd.DataPath)
	if err := root.ExecuteContext(ctx); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "[!] %v\n", err)
		os.Exit(1)
	}
}
