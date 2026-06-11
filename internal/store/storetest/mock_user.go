package storetest

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"nucleus/internal/apperror"
	"nucleus/internal/store"
)

// MockUserStore 是 UserStore 的内存实现，用于测试。
type MockUserStore struct {
	users  map[string]store.User
	nextID int64

	CreatedUsers []store.User

	CreateErr     error
	GetByEmailErr error
}

func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		users:  make(map[string]store.User),
		nextID: 1,
	}
}

func (m *MockUserStore) Create(_ context.Context, input store.CreateUserInput) (store.User, error) {
	if m.CreateErr != nil {
		return store.User{}, m.CreateErr
	}
	if _, exists := m.users[input.Email]; exists {
		return store.User{}, apperror.Conflict("user already exists")
	}

	now := time.Now().Truncate(time.Second)
	user := store.User{
		ID:           m.nextID,
		Email:        input.Email,
		Name:         input.Name,
		PasswordHash: input.PasswordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	m.nextID++
	m.users[user.Email] = user
	m.CreatedUsers = append(m.CreatedUsers, user)
	return user, nil
}

func (m *MockUserStore) GetByEmail(_ context.Context, email string) (store.User, error) {
	if m.GetByEmailErr != nil {
		return store.User{}, m.GetByEmailErr
	}
	user, exists := m.users[email]
	if !exists {
		return store.User{}, apperror.NotFound("user not found")
	}
	return user, nil
}

func (m *MockUserStore) WithTx(pgx.Tx) store.UserStore {
	return m
}
