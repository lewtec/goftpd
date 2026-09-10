package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lucasew/goftpd"
)

type flags struct {
	Addr cmd.StringArg `long:"addr" help:"Listen address"`
	Dir  cmd.StringArg `short:"d" long:"dir" help:"Served directory"`
	SPA  cmd.Flag      `long:"spa" help:"SPA mode: never list directories"`
}

func (f flags) Run(ctx context.Context) error {
	app, err := goftpd.NewApp(goftpd.Config{
		Addr: f.Addr.Value(),
		Dir:  f.Dir.Value(),
		SPA:  f.SPA.Value(),
	})
	if err != nil {
		return err
	}
	return app.Run(ctx)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		slog.Error("goftpd", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	app, err := cmd.Parse[cmd.App[flags]](args...)
	if err != nil {
		return err
	}
	return app.Run(ctx)
}
