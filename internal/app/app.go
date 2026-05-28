package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"nucleus/internal/config"
	"nucleus/internal/db"
	"nucleus/internal/db/sqlc"
	"nucleus/internal/http/handler"
	"nucleus/internal/http/router"
)

type App struct {
	cfg    config.Config
	pool   *db.Pool
	server *http.Server
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	pool, err := db.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	queries := sqlc.New(pool.DB())
	healthHandler := handler.NewHealthHandler()
	noteHandler := handler.NewNoteHandler(queries)

	server := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           router.New(healthHandler, noteHandler),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	return &App{
		cfg:    cfg,
		pool:   pool,
		server: server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("server listening on :%s (%s)", a.cfg.HTTP.Port, a.cfg.App.Env)
		errCh <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown signal received")
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			a.pool.Close()
			return fmt.Errorf("server failed: %w", err)
		}
	}

	return a.shutdown()
}

func (a *App) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		a.pool.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	a.pool.Close()
	log.Println("server stopped gracefully")
	return nil
}
