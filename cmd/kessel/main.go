// Command kessel is the Kessel website performance monitor.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	webembed "github.com/t0mer/kessel"
	"github.com/t0mer/kessel/internal/app"
	"github.com/t0mer/kessel/internal/config"
	"github.com/t0mer/kessel/internal/logging"
	"github.com/t0mer/kessel/internal/version"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "kessel",
		Short:   "Website performance monitoring via PageSpeed Insights",
		Version: version.Version,
	}
	root.SetVersionTemplate("kessel {{.Version}}\n")

	serve := &cobra.Command{
		Use:   "serve",
		Short: "Run the Kessel server",
		RunE:  runServe,
	}
	config.RegisterFlags(serve)
	root.AddCommand(serve)
	return root
}

func runServe(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(cmd)
	if err != nil {
		return err
	}
	log := logging.New(cfg.LogLevel, cfg.LogFormat)

	dist, err := webembed.FS()
	if err != nil {
		return fmt.Errorf("loading embedded UI: %w", err)
	}

	application, err := app.New(cfg, log, dist)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Start(ctx); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() { errCh <- application.Serve() }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return application.Shutdown(shutCtx)
	}
}

// Execute runs the root command.
func Execute() error { return newRootCmd().Execute() }

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
