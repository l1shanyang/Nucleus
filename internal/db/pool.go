package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"nucleus/internal/config"
)

type Pool struct {
	inner *pgxpool.Pool
}

func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MinConns)
	poolCfg.MaxConnIdleTime = cfg.MaxIdleTime
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	return &Pool{inner: pool}, nil
}

func (p *Pool) Ping(ctx context.Context) error {
	return p.inner.Ping(ctx)
}

func (p *Pool) Close() {
	p.inner.Close()
}

func (p *Pool) DB() *pgxpool.Pool {
	return p.inner
}

// BeginTx 开启一个数据库事务。
func (p *Pool) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return p.inner.Begin(ctx)
}
