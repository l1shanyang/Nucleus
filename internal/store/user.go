package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"nucleus/internal/db/sqlc"
)

// User 是 store 层的用户模型。PasswordHash 只能在业务层内部使用，不能序列化给客户端。
type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateUserInput struct {
	Email        string
	Name         string
	PasswordHash string
}

// UserStore 定义用户数据访问接口。
type UserStore interface {
	Create(ctx context.Context, input CreateUserInput) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	WithTx(tx pgx.Tx) UserStore
}

type userStore struct {
	q *sqlc.Queries
}

func NewUserStore(q *sqlc.Queries) UserStore {
	return &userStore{q: q}
}

func (s *userStore) Create(ctx context.Context, input CreateUserInput) (User, error) {
	row, err := s.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        input.Email,
		Name:         input.Name,
		PasswordHash: input.PasswordHash,
	})
	if err != nil {
		return User{}, mapDBError(err, "user")
	}
	return toUser(&row), nil
}

func (s *userStore) GetByEmail(ctx context.Context, email string) (User, error) {
	row, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, mapDBError(err, "user")
	}
	return toUser(&row), nil
}

func (s *userStore) WithTx(tx pgx.Tx) UserStore {
	return &userStore{q: s.q.WithTx(tx)}
}

func toUser(row *sqlc.User) User {
	return User{
		ID:           row.ID,
		Email:        row.Email,
		Name:         row.Name,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
