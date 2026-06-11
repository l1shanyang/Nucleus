package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
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
	sessionTokenBytes = 32
	sessionTTL        = 24 * time.Hour
)

// AuthService 封装认证相关业务逻辑。
type AuthService struct {
	users    store.UserStore
	sessions store.SessionStore
}

func NewAuthService(users store.UserStore, sessions store.SessionStore) *AuthService {
	return &AuthService{users: users, sessions: sessions}
}

type RegisterInput struct {
	Email    string
	Name     string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

// AuthUser 是认证接口返回给客户端的用户公开信息。
type AuthUser struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type AuthToken struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        AuthUser  `json:"user"`
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

func (s *AuthService) Login(ctx context.Context, input LoginInput) (AuthToken, error) {
	email := normalizeEmail(input.Email)
	if err := validateLoginInput(email, input.Password); err != nil {
		return AuthToken{}, err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return AuthToken{}, mapLoginUserError(err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return AuthToken{}, apperror.Unauthorized("invalid email or password", err)
	}

	token, err := generateSessionToken()
	if err != nil {
		return AuthToken{}, apperror.Internal("failed to generate session token", err)
	}

	expiresAt := time.Now().UTC().Add(sessionTTL)
	if _, err := s.sessions.Create(ctx, store.CreateSessionInput{
		UserID:    user.ID,
		TokenHash: hashSessionToken(token),
		ExpiresAt: expiresAt,
	}); err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) {
			return AuthToken{}, err
		}
		return AuthToken{}, apperror.Internal("failed to create auth session", err)
	}

	return AuthToken{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        toAuthUser(&user),
	}, nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (AuthUser, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return AuthUser{}, apperror.Unauthorized("authorization bearer token is required")
	}

	user, err := s.sessions.GetUserByTokenHash(ctx, hashSessionToken(token))
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) && appErr.Kind == apperror.KindNotFound {
			return AuthUser{}, apperror.Unauthorized("invalid or expired token", err)
		}
		if errors.As(err, &appErr) {
			return AuthUser{}, err
		}
		return AuthUser{}, apperror.Internal("failed to authenticate token", err)
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

func validateLoginInput(email, password string) error {
	if email == "" {
		return apperror.Validation("email is required")
	}
	if len(email) > maxEmailLength {
		return apperror.Validation("email must be at most 320 characters")
	}
	if !isValidEmail(email) {
		return apperror.Validation("email is invalid")
	}
	if strings.TrimSpace(password) == "" {
		return apperror.Validation("password is required")
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

func mapLoginUserError(err error) error {
	var appErr *apperror.Error
	if errors.As(err, &appErr) && appErr.Kind == apperror.KindNotFound {
		return apperror.Unauthorized("invalid email or password", err)
	}
	if errors.As(err, &appErr) {
		return err
	}
	return apperror.Internal("failed to load user", err)
}

func generateSessionToken() (string, error) {
	b := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
