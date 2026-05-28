package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"nucleus/internal/app"
	"nucleus/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	a, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	if err := a.Run(ctx); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
