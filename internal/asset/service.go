package asset

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/nicolastomaschewski/controle-financeiro/internal/audit"
)

type Service struct {
	repo  *Repository
	audit *audit.Recorder
}

func NewService(repo *Repository, audit *audit.Recorder) *Service {
	return &Service{repo: repo, audit: audit}
}

type CreateInput struct {
	Name          string
	Type          string
	CurrentValue  decimal.Decimal
	ValuationDate time.Time
	Notes         *string
}

func (s *Service) Create(ctx context.Context, userID string, in CreateInput) (*Asset, error) {
	if err := ValidateName(in.Name); err != nil {
		return nil, err
	}
	if err := ValidateType(in.Type); err != nil {
		return nil, err
	}
	if err := ValidateValue(in.CurrentValue); err != nil {
		return nil, err
	}

	a := &Asset{
		UserID: userID, Name: in.Name, Type: in.Type, CurrentValue: in.CurrentValue,
		ValuationDate: in.ValuationDate, Notes: in.Notes, Status: StatusActive,
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("criar ativo: %w", err)
	}
	s.audit.Record(ctx, &userID, "asset.created", "asset", a.ID, nil)
	return a, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (*Asset, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *Service) List(ctx context.Context, userID string, includeArchived bool) ([]*Asset, error) {
	return s.repo.List(ctx, userID, includeArchived)
}

type UpdateInput struct {
	Name   string
	Type   string
	Notes  *string
	Status string
}

func (s *Service) Update(ctx context.Context, userID, id string, in UpdateInput) (*Asset, error) {
	if err := ValidateName(in.Name); err != nil {
		return nil, err
	}
	if err := ValidateType(in.Type); err != nil {
		return nil, err
	}
	if in.Status != StatusActive && in.Status != StatusArchived {
		return nil, fmt.Errorf("status inválido: %s", in.Status)
	}
	if err := s.repo.UpdateDetails(ctx, id, userID, in.Name, in.Type, in.Notes, in.Status); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "asset.updated", "asset", id, nil)
	return s.repo.GetByID(ctx, id, userID)
}

// AddValuation registra uma nova avaliação (manual, informada pelo usuário)
// para o ativo, atualizando seu valor atual sem apagar o histórico.
func (s *Service) AddValuation(ctx context.Context, userID, assetID string, value decimal.Decimal, date time.Time, notes *string) (*Valuation, error) {
	if err := ValidateValue(value); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByID(ctx, assetID, userID); err != nil {
		return nil, err
	}
	v, err := s.repo.AddValuation(ctx, assetID, userID, value, date, notes)
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "asset.valuation_added", "asset", assetID, map[string]any{"value": value.String(), "date": date.Format("2006-01-02")})
	return v, nil
}

func (s *Service) ListValuations(ctx context.Context, userID, assetID string) ([]*Valuation, error) {
	if _, err := s.repo.GetByID(ctx, assetID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListValuations(ctx, assetID, userID)
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	s.audit.Record(ctx, &userID, "asset.deleted", "asset", id, nil)
	return nil
}
