package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"nucleus/internal/app"
	"nucleus/internal/config"
	applog "nucleus/internal/log"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "error", err)
		return
	}

	applog.Setup(cfg.App.Env, cfg.Log.Level)

	a, err := app.New(ctx, cfg)
	if err != nil {
		slog.Error("init app failed", "error", err)
		return
	}

	if err := a.Run(ctx); err != nil {
		slog.Error("run app failed", "error", err)
	}
}
