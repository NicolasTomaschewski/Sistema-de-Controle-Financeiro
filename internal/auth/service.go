package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nicolastomaschewski/controle-financeiro/internal/audit"
	"github.com/nicolastomaschewski/controle-financeiro/internal/user"
)

var (
	ErrInvalidCredentials = errors.New("e-mail ou senha inválidos")
	ErrRateLimited        = errors.New("muitas tentativas de login; tente novamente mais tarde")
)

type Service struct {
	users         *user.Repository
	sessions      *SessionRepository
	loginAttempts *LoginAttemptRepository
	audit         *audit.Recorder
}

func NewService(users *user.Repository, sessions *SessionRepository, loginAttempts *LoginAttemptRepository, audit *audit.Recorder) *Service {
	return &Service{users: users, sessions: sessions, loginAttempts: loginAttempts, audit: audit}
}

// Register cadastra um novo usuário com senha protegida por hashing.
// Não realiza login automático nem emite sessão — o cliente deve autenticar
// explicitamente após o cadastro.
func (s *Service) Register(ctx context.Context, name, email, password string) (*user.User, error) {
	if err := user.ValidateName(name); err != nil {
		return nil, err
	}
	if err := user.ValidateEmail(email); err != nil {
		return nil, err
	}
	if err := user.ValidatePassword(password); err != nil {
		return nil, err
	}

	hash, err := user.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("gerar hash de senha: %w", err)
	}

	u := &user.User{
		Name:            name,
		Email:           email,
		EmailNormalized: user.NormalizeEmail(email),
		PasswordHash:    hash,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}

	s.audit.Record(ctx, &u.ID, "user.registered", "user", u.ID, nil)
	return u, nil
}

// Login autentica um usuário, aplicando limite de tentativas persistente e
// retornando uma mensagem genérica em caso de falha, para não permitir
// enumerar contas existentes.
func (s *Service) Login(ctx context.Context, email, password, ip, userAgent string) (token string, expiresAt time.Time, err error) {
	normalized := user.NormalizeEmail(email)

	limited, err := s.loginAttempts.IsRateLimited(ctx, normalized, ip)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("verificar limite de tentativas: %w", err)
	}
	if limited {
		return "", time.Time{}, ErrRateLimited
	}

	u, lookupErr := s.users.GetByEmailNormalized(ctx, normalized)
	if lookupErr != nil {
		// Mesmo que o usuário não exista, registra a tentativa sob o e-mail
		// informado para que o limite de tentativas também se aplique a
		// ataques de enumeração de contas inexistentes.
		_ = s.loginAttempts.Record(ctx, normalized, ip, false)
		return "", time.Time{}, ErrInvalidCredentials
	}

	ok, verifyErr := user.VerifyPassword(u.PasswordHash, password)
	if verifyErr != nil || !ok {
		_ = s.loginAttempts.Record(ctx, normalized, ip, false)
		return "", time.Time{}, ErrInvalidCredentials
	}

	_ = s.loginAttempts.Record(ctx, normalized, ip, true)

	token, expiresAt, err = s.sessions.Create(ctx, u.ID, userAgent)
	if err != nil {
		return "", time.Time{}, err
	}

	s.audit.Record(ctx, &u.ID, "user.login", "user", u.ID, nil)
	return token, expiresAt, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.sessions.Revoke(ctx, token)
}
