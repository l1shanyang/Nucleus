package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"nucleus/internal/db"
)

func TestTxManager_WithTx_CommitsOnSuccess(t *testing.T) {
	tx := &fakeTx{}
	starter := &fakeTxStarter{tx: tx}
	manager := db.NewTxManager(starter)

	err := manager.WithTx(context.Background(), func(got pgx.Tx) error {
		if got != tx {
			t.Fatal("callback received unexpected tx")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if starter.beginCount != 1 {
		t.Fatalf("begin count = %d, want 1", starter.beginCount)
	}
	if !tx.committed {
		t.Fatal("expected tx to be committed")
	}
	if tx.rolledBack {
		t.Fatal("did not expect tx to be rolled back")
	}
}

func TestTxManager_WithTx_RollsBackOnCallbackError(t *testing.T) {
	cause := errors.New("create order failed")
	tx := &fakeTx{}
	manager := db.NewTxManager(&fakeTxStarter{tx: tx})

	err := manager.WithTx(context.Background(), func(pgx.Tx) error {
		return cause
	})

	if !errors.Is(err, cause) {
		t.Fatalf("error = %v, want cause %v", err, cause)
	}
	if tx.committed {
		t.Fatal("did not expect tx to be committed")
	}
	if !tx.rolledBack {
		t.Fatal("expected tx to be rolled back")
	}
}

func TestTxManager_WithTx_ReturnsBeginError(t *testing.T) {
	cause := errors.New("database unavailable")
	manager := db.NewTxManager(&fakeTxStarter{err: cause})

	err := manager.WithTx(context.Background(), func(pgx.Tx) error {
		t.Fatal("callback should not run when begin fails")
		return nil
	})

	if !errors.Is(err, cause) {
		t.Fatalf("error = %v, want cause %v", err, cause)
	}
}

func TestTxManager_WithTx_RollsBackOnCommitError(t *testing.T) {
	cause := errors.New("commit failed")
	tx := &fakeTx{commitErr: cause}
	manager := db.NewTxManager(&fakeTxStarter{tx: tx})

	err := manager.WithTx(context.Background(), func(pgx.Tx) error {
		return nil
	})

	if !errors.Is(err, cause) {
		t.Fatalf("error = %v, want cause %v", err, cause)
	}
	if !tx.committed {
		t.Fatal("expected commit to be attempted")
	}
	if !tx.rolledBack {
		t.Fatal("expected rollback after commit failure")
	}
}

type fakeTxStarter struct {
	tx         pgx.Tx
	err        error
	beginCount int
}

func (s *fakeTxStarter) BeginTx(context.Context) (pgx.Tx, error) {
	s.beginCount++
	if s.err != nil {
		return nil, s.err
	}
	return s.tx, nil
}

type fakeTx struct {
	pgx.Tx

	committed  bool
	rolledBack bool
	commitErr  error
}

func (tx *fakeTx) Commit(context.Context) error {
	tx.committed = true
	return tx.commitErr
}

func (tx *fakeTx) Rollback(context.Context) error {
	tx.rolledBack = true
	return nil
}
