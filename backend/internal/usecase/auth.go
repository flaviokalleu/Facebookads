package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
)

type AuthUseCase struct {
	users repository.UserRepository
	cfg   *config.Service
}

func NewAuthUseCase(users repository.UserRepository, cfg *config.Service) *AuthUseCase {
	return &AuthUseCase{users: users, cfg: cfg}
}

type RegisterInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
}

type LoginInput struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type AuthOutput struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      *domain.User `json:"user"`
}

func (uc *AuthUseCase) Register(ctx context.Context, in RegisterInput) (*AuthOutput, error) {
	existing, _ := uc.users.GetByEmail(ctx, in.Email)
	if existing != nil {
		return nil, fmt.Errorf("%w: email already registered", domain.ErrConflict)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Email:        in.Email,
		PasswordHash: string(hash),
		Name:         in.Name,
	}
	if err := uc.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return uc.issueToken(user)
}

func (uc *AuthUseCase) Login(ctx context.Context, in LoginInput) (*AuthOutput, error) {
	user, err := uc.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid credentials", domain.ErrUnauthorized)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, fmt.Errorf("%w: invalid credentials", domain.ErrUnauthorized)
	}

	return uc.issueToken(user)
}

func (uc *AuthUseCase) GetUser(ctx context.Context, id string) (*domain.User, error) {
	user, err := uc.users.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	user.PasswordHash = ""
	return user, nil
}

func (uc *AuthUseCase) issueToken(user *domain.User) (*AuthOutput, error) {
	secret := uc.cfg.GetSecret("jwt.secret")
	if secret == "" {
		return nil, fmt.Errorf("jwt.secret not configured")
	}

	exp := time.Now().Add(7 * 24 * time.Hour)
	claims := middleware.Claims{
		UserID:  user.ID,
		IsAdmin: user.IsAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}

	user.PasswordHash = "" // never expose
	return &AuthOutput{Token: signed, ExpiresAt: exp, User: user}, nil
}
