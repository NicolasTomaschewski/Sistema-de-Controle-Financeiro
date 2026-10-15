package user

import (
	"context"
	"fmt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetProfile retorna o perfil do usuário autenticado. O id deve vir sempre
// da sessão validada, nunca de um parâmetro de requisição.
func (s *Service) GetProfile(ctx context.Context, userID string) (*User, error) {
	return s.repo.GetByID(ctx, userID)
}

// UpdateProfile atualiza nome e e-mail do próprio usuário autenticado.
func (s *Service) UpdateProfile(ctx context.Context, userID, name, email string) (*User, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := ValidateEmail(email); err != nil {
		return nil, err
	}
	normalized := NormalizeEmail(email)

	if err := s.repo.UpdateProfile(ctx, userID, name, email, normalized); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, userID)
}

// ChangePassword exige a senha atual correta antes de permitir a troca,
// evitando que uma sessão sequestrada temporariamente baste para assumir a
// conta de forma permanente.
func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	ok, err := VerifyPassword(u.PasswordHash, currentPassword)
	if err != nil {
		return fmt.Errorf("verificar senha atual: %w", err)
	}
	if !ok {
		return ErrWrongPassword
	}

	newHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("gerar novo hash de senha: %w", err)
	}

	return s.repo.UpdatePasswordHash(ctx, userID, newHash)
}
