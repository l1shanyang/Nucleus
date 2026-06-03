package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// TxStarter 是可以开启数据库事务的最小接口。
type TxStarter interface {
	BeginTx(ctx context.Context) (pgx.Tx, error)
}

// TxManager 统一管理事务的 begin / commit / rollback。
type TxManager struct {
	starter TxStarter
}

func NewTxManager(starter TxStarter) *TxManager {
	return &TxManager{starter: starter}
}

// WithTx 在一个事务中执行 fn。
func (m *TxManager) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) (err error) {
	tx, err := m.starter.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && err == nil {
			err = fmt.Errorf("rollback tx: %w", rollbackErr)
		}
	}()

	if fnErr := fn(tx); fnErr != nil {
		err = fnErr
		return err
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		err = fmt.Errorf("commit tx: %w", commitErr)
		return err
	}
	committed = true
	return nil
}
