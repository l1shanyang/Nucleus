package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"nucleus/internal/apperror"
	"nucleus/internal/store"
)

const (
	maxEmailLength    = 320
	maxNameLength     = 100
	minPasswordLength = 8
	maxBcryptPassword = 72
)

// AuthService 封装认证相关业务逻辑。
type AuthService struct {
	users store.UserStore
}

func NewAuthService(users store.UserStore) *AuthService {
	return &AuthService{users: users}
}

type RegisterInput struct {
	Email    string
	Name     string
	Password string
}

// AuthUser 是认证接口返回给客户端的用户公开信息。
type AuthUser struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (AuthUser, error) {
	email := normalizeEmail(input.Email)
	name := strings.TrimSpace(input.Name)

	if err := validateRegisterInput(email, name, input.Password); err != nil {
		return AuthUser{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthUser{}, apperror.Internal("failed to hash password", err)
	}

	user, err := s.users.Create(ctx, store.CreateUserInput{
		Email:        email,
		Name:         name,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) {
			return AuthUser{}, err
		}
		return AuthUser{}, apperror.Internal("failed to register user", err)
	}

	return toAuthUser(&user), nil
}

func validateRegisterInput(email, name, password string) error {
	if email == "" {
		return apperror.Validation("email is required")
	}
	if len(email) > maxEmailLength {
		return apperror.Validation("email must be at most 320 characters")
	}
	if !isValidEmail(email) {
		return apperror.Validation("email is invalid")
	}
	if name == "" {
		return apperror.Validation("name is required")
	}
	if len(name) > maxNameLength {
		return apperror.Validation("name must be at most 100 characters")
	}
	if strings.TrimSpace(password) == "" {
		return apperror.Validation("password is required")
	}
	if len(password) < minPasswordLength {
		return apperror.Validation("password must be at least 8 characters")
	}
	if len(password) > maxBcryptPassword {
		return apperror.Validation("password must be at most 72 bytes")
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func isValidEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}

func toAuthUser(user *store.User) AuthUser {
	return AuthUser{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
	}
}
