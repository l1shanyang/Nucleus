package storetest

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"nucleus/internal/apperror"
	"nucleus/internal/store"
)

type MockSessionStore struct {
	sessions map[string]store.AuthSession
	users    map[string]store.User
	nextID   int64

	CreatedSessions []store.AuthSession

	CreateErr             error
	GetUserByTokenHashErr error
}

func NewMockSessionStore() *MockSessionStore {
	return &MockSessionStore{
		sessions: make(map[string]store.AuthSession),
		users:    make(map[string]store.User),
		nextID:   1,
	}
}

func (m *MockSessionStore) Create(_ context.Context, input store.CreateSessionInput) (store.AuthSession, error) {
	if m.CreateErr != nil {
		return store.AuthSession{}, m.CreateErr
	}

	session := store.AuthSession{
		ID:        m.nextID,
		UserID:    input.UserID,
		TokenHash: input.TokenHash,
		ExpiresAt: input.ExpiresAt,
		CreatedAt: time.Now().Truncate(time.Second),
	}
	m.nextID++
	m.sessions[session.TokenHash] = session
	m.CreatedSessions = append(m.CreatedSessions, session)
	return session, nil
}

func (m *MockSessionStore) GetUserByTokenHash(_ context.Context, tokenHash string) (store.User, error) {
	if m.GetUserByTokenHashErr != nil {
		return store.User{}, m.GetUserByTokenHashErr
	}
	session, exists := m.sessions[tokenHash]
	if !exists || time.Now().After(session.ExpiresAt) {
		return store.User{}, apperror.NotFound("auth session not found")
	}
	user, exists := m.users[tokenHash]
	if !exists {
		return store.User{}, apperror.NotFound("auth session not found")
	}
	return user, nil
}

func (m *MockSessionStore) WithTx(pgx.Tx) store.SessionStore {
	return m
}

func (m *MockSessionStore) AttachUser(tokenHash string, user *store.User) {
	m.users[tokenHash] = *user
}
