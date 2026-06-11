package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"nucleus/internal/apperror"
	"nucleus/internal/service"
	"nucleus/internal/store/storetest"
)

func setupAuthService() (*service.AuthService, *storetest.MockUserStore, *storetest.MockSessionStore) {
	users := storetest.NewMockUserStore()
	sessions := storetest.NewMockSessionStore()
	svc := service.NewAuthService(users, sessions)
	return svc, users, sessions
}

func TestAuthService_Register_Success(t *testing.T) {
	svc, users, _ := setupAuthService()

	user, err := svc.Register(context.Background(), service.RegisterInput{
		Email:    "  USER@example.COM  ",
		Name:     "  Alice  ",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID == 0 {
		t.Fatal("expected non-zero user ID")
	}
	if user.Email != "user@example.com" {
		t.Fatalf("email = %q, want user@example.com", user.Email)
	}
	if user.Name != "Alice" {
		t.Fatalf("name = %q, want Alice", user.Name)
	}
	if len(users.CreatedUsers) != 1 {
		t.Fatalf("created users = %d, want 1", len(users.CreatedUsers))
	}

	hash := users.CreatedUsers[0].PasswordHash
	if hash == "password123" {
		t.Fatal("password was stored as plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("password123")); err != nil {
		t.Fatalf("stored password hash is invalid: %v", err)
	}
}

func TestAuthService_Register_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   service.RegisterInput
		wantErr string
	}{
		{
			name:    "email 为空",
			input:   service.RegisterInput{Email: "", Name: "Alice", Password: "password123"},
			wantErr: "email is required",
		},
		{
			name:    "email 格式错误",
			input:   service.RegisterInput{Email: "not-email", Name: "Alice", Password: "password123"},
			wantErr: "email is invalid",
		},
		{
			name:    "name 为空",
			input:   service.RegisterInput{Email: "user@example.com", Name: " ", Password: "password123"},
			wantErr: "name is required",
		},
		{
			name:    "password 为空",
			input:   service.RegisterInput{Email: "user@example.com", Name: "Alice", Password: "   "},
			wantErr: "password is required",
		},
		{
			name:    "password 过短",
			input:   service.RegisterInput{Email: "user@example.com", Name: "Alice", Password: "short"},
			wantErr: "password must be at least 8 characters",
		},
		{
			name:    "password 超过 bcrypt 上限",
			input:   service.RegisterInput{Email: "user@example.com", Name: "Alice", Password: strings.Repeat("a", 73)},
			wantErr: "password must be at most 72 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, users, _ := setupAuthService()

			_, err := svc.Register(context.Background(), tt.input)
			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}

			var appErr *apperror.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("expected apperror.Error, got %T", err)
			}
			if appErr.Kind != apperror.KindValidation {
				t.Fatalf("kind = %q, want %q", appErr.Kind, apperror.KindValidation)
			}
			if appErr.Message != tt.wantErr {
				t.Fatalf("message = %q, want %q", appErr.Message, tt.wantErr)
			}
			if len(users.CreatedUsers) != 0 {
				t.Fatalf("invalid input should not create users")
			}
		})
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	svc, _, _ := setupAuthService()

	input := service.RegisterInput{Email: "user@example.com", Name: "Alice", Password: "password123"}
	if _, err := svc.Register(context.Background(), input); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	_, err := svc.Register(context.Background(), input)
	if err == nil {
		t.Fatal("expected duplicate email error, got nil")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected apperror.Error, got %T", err)
	}
	if appErr.Kind != apperror.KindConflict {
		t.Fatalf("kind = %q, want %q", appErr.Kind, apperror.KindConflict)
	}
	if appErr.Message != "user already exists" {
		t.Fatalf("message = %q, want user already exists", appErr.Message)
	}
}

func TestAuthService_Register_StoreError(t *testing.T) {
	svc, users, _ := setupAuthService()
	users.CreateErr = &testError{"database unavailable"}

	_, err := svc.Register(context.Background(), service.RegisterInput{
		Email:    "user@example.com",
		Name:     "Alice",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected apperror.Error, got %T", err)
	}
	if appErr.Kind != apperror.KindInternal {
		t.Fatalf("kind = %q, want %q", appErr.Kind, apperror.KindInternal)
	}
	if !errors.Is(err, users.CreateErr) {
		t.Fatal("expected service error to wrap store error")
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	svc, _, sessions := setupAuthService()

	registered, err := svc.Register(context.Background(), service.RegisterInput{
		Email:    "user@example.com",
		Name:     "Alice",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	token, err := svc.Login(context.Background(), service.LoginInput{
		Email:    "USER@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if token.AccessToken == "" {
		t.Fatal("expected access token")
	}
	if token.TokenType != "Bearer" {
		t.Fatalf("token type = %q, want Bearer", token.TokenType)
	}
	if !token.ExpiresAt.After(timeNow()) {
		t.Fatalf("expected future expiration, got %s", token.ExpiresAt)
	}
	if token.User.ID != registered.ID {
		t.Fatalf("user id = %d, want %d", token.User.ID, registered.ID)
	}
	if len(sessions.CreatedSessions) != 1 {
		t.Fatalf("created sessions = %d, want 1", len(sessions.CreatedSessions))
	}
	session := sessions.CreatedSessions[0]
	if session.UserID != registered.ID {
		t.Fatalf("session user id = %d, want %d", session.UserID, registered.ID)
	}
	if session.TokenHash == token.AccessToken {
		t.Fatal("session token hash should not store raw access token")
	}
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	svc, _, _ := setupAuthService()
	if _, err := svc.Register(context.Background(), service.RegisterInput{
		Email:    "user@example.com",
		Name:     "Alice",
		Password: "password123",
	}); err != nil {
		t.Fatalf("register user: %v", err)
	}

	tests := []struct {
		name  string
		input service.LoginInput
	}{
		{
			name:  "邮箱不存在",
			input: service.LoginInput{Email: "missing@example.com", Password: "password123"},
		},
		{
			name:  "密码错误",
			input: service.LoginInput{Email: "user@example.com", Password: "wrong-password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Login(context.Background(), tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var appErr *apperror.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("expected apperror.Error, got %T", err)
			}
			if appErr.Kind != apperror.KindUnauthorized {
				t.Fatalf("kind = %q, want %q", appErr.Kind, apperror.KindUnauthorized)
			}
			if appErr.Message != "invalid email or password" {
				t.Fatalf("message = %q, want invalid email or password", appErr.Message)
			}
		})
	}
}

func TestAuthService_Authenticate(t *testing.T) {
	svc, users, sessions := setupAuthService()

	if _, err := svc.Register(context.Background(), service.RegisterInput{
		Email:    "user@example.com",
		Name:     "Alice",
		Password: "password123",
	}); err != nil {
		t.Fatalf("register user: %v", err)
	}

	token, err := svc.Login(context.Background(), service.LoginInput{
		Email:    "user@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if len(sessions.CreatedSessions) != 1 || len(users.CreatedUsers) != 1 {
		t.Fatal("expected one user and one session")
	}
	sessions.AttachUser(sessions.CreatedSessions[0].TokenHash, &users.CreatedUsers[0])

	user, err := svc.Authenticate(context.Background(), token.AccessToken)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Fatalf("email = %q, want user@example.com", user.Email)
	}

	_, err = svc.Authenticate(context.Background(), "invalid-token")
	if err == nil {
		t.Fatal("expected invalid token error, got nil")
	}
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected apperror.Error, got %T", err)
	}
	if appErr.Kind != apperror.KindUnauthorized {
		t.Fatalf("kind = %q, want %q", appErr.Kind, apperror.KindUnauthorized)
	}
}

func timeNow() time.Time {
	return time.Now()
}
