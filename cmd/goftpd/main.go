package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lucasew/goftpd"
	"github.com/spf13/cobra"
)

// Set via goreleaser ldflags: -X main.version={{ .Version }}
var version = "dev"

func main() {
	var cfg goftpd.Config

	cmd := &cobra.Command{
		Use:           "goftpd",
		Version:       version,
		Short:         "Simple HTTP file server",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := goftpd.NewApp(cfg)
			if err != nil {
				if uerr := cmd.Usage(); uerr != nil {
					return errors.Join(err, uerr)
				}
				return err
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return app.Run(ctx)
		},
	}
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.Flags().StringVar(&cfg.Addr, "addr", ":8080", "Listen address")
	cmd.Flags().StringVarP(&cfg.Dir, "dir", "d", "./", "Served directory")
	cmd.Flags().BoolVar(&cfg.SPA, "spa", false, "SPA mode: never list directories")

	if err := cmd.Execute(); err != nil {
		slog.Error("goftpd", "err", err)
		os.Exit(1)
	}
}
