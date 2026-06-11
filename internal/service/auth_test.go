package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"nucleus/internal/apperror"
	"nucleus/internal/service"
	"nucleus/internal/store/storetest"
)

func TestAuthService_Register_Success(t *testing.T) {
	mock := storetest.NewMockUserStore()
	svc := service.NewAuthService(mock)

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
	if len(mock.CreatedUsers) != 1 {
		t.Fatalf("created users = %d, want 1", len(mock.CreatedUsers))
	}

	hash := mock.CreatedUsers[0].PasswordHash
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
			mock := storetest.NewMockUserStore()
			svc := service.NewAuthService(mock)

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
			if len(mock.CreatedUsers) != 0 {
				t.Fatalf("invalid input should not create users")
			}
		})
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	mock := storetest.NewMockUserStore()
	svc := service.NewAuthService(mock)

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
	mock := storetest.NewMockUserStore()
	mock.CreateErr = &testError{"database unavailable"}
	svc := service.NewAuthService(mock)

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
	if !errors.Is(err, mock.CreateErr) {
		t.Fatal("expected service error to wrap store error")
	}
}
