package category

import (
	"context"

	"github.com/nicolastomaschewski/controle-financeiro/internal/audit"
)

type Service struct {
	repo  *Repository
	audit *audit.Recorder
}

func NewService(repo *Repository, audit *audit.Recorder) *Service {
	return &Service{repo: repo, audit: audit}
}

func (s *Service) Create(ctx context.Context, userID, name, kind string) (*Category, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	if err := ValidateKind(kind); err != nil {
		return nil, err
	}
	c, err := s.repo.Create(ctx, userID, name, kind)
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "category.created", "category", c.ID, nil)
	return c, nil
}

func (s *Service) List(ctx context.Context, userID string, includeArchived bool) ([]*Category, error) {
	return s.repo.ListVisible(ctx, userID, includeArchived)
}

func (s *Service) Update(ctx context.Context, userID, id, name string) (*Category, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetVisible(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if existing.IsSystem() {
		return nil, ErrSystemCategory
	}

	if err := s.repo.UpdateName(ctx, id, userID, name); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "category.updated", "category", id, nil)
	return s.repo.GetVisible(ctx, id, userID)
}

// Delete exclui a categoria privada do usuário se ela não estiver em uso;
// caso contrário, arquiva-a para preservar a integridade dos lançamentos
// históricos. Categorias globais do sistema nunca podem ser removidas por
// um usuário.
func (s *Service) Delete(ctx context.Context, userID, id string) (archived bool, err error) {
	existing, err := s.repo.GetVisible(ctx, id, userID)
	if err != nil {
		return false, err
	}
	if existing.IsSystem() {
		return false, ErrSystemCategory
	}

	inUse, err := s.repo.InUse(ctx, id)
	if err != nil {
		return false, err
	}
	if inUse {
		if err := s.repo.Archive(ctx, id, userID); err != nil {
			return false, err
		}
		s.audit.Record(ctx, &userID, "category.archived", "category", id, nil)
		return true, nil
	}

	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return false, err
	}
	s.audit.Record(ctx, &userID, "category.deleted", "category", id, nil)
	return false, nil
}
