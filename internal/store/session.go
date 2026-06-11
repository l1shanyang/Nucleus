package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"nucleus/internal/db/sqlc"
)

type AuthSession struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type CreateSessionInput struct {
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
}

type SessionStore interface {
	Create(ctx context.Context, input CreateSessionInput) (AuthSession, error)
	GetUserByTokenHash(ctx context.Context, tokenHash string) (User, error)
	WithTx(tx pgx.Tx) SessionStore
}

type sessionStore struct {
	q *sqlc.Queries
}

func NewSessionStore(q *sqlc.Queries) SessionStore {
	return &sessionStore{q: q}
}

func (s *sessionStore) Create(ctx context.Context, input CreateSessionInput) (AuthSession, error) {
	row, err := s.q.CreateAuthSession(ctx, sqlc.CreateAuthSessionParams{
		UserID:    input.UserID,
		TokenHash: input.TokenHash,
		ExpiresAt: input.ExpiresAt,
	})
	if err != nil {
		return AuthSession{}, mapDBError(err, "auth session")
	}
	return toAuthSession(&row), nil
}

func (s *sessionStore) GetUserByTokenHash(ctx context.Context, tokenHash string) (User, error) {
	row, err := s.q.GetAuthSessionUserByTokenHash(ctx, tokenHash)
	if err != nil {
		return User{}, mapDBError(err, "auth session")
	}
	return toUser(&row), nil
}

func (s *sessionStore) WithTx(tx pgx.Tx) SessionStore {
	return &sessionStore{q: s.q.WithTx(tx)}
}

func toAuthSession(row *sqlc.AuthSession) AuthSession {
	return AuthSession{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}
}
