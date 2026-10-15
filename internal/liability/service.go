package liability

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
	Name            string
	Type            string
	PrincipalAmount decimal.Decimal
	CurrentBalance  decimal.Decimal
	ReferenceDate   time.Time
	Notes           *string
}

func (s *Service) Create(ctx context.Context, userID string, in CreateInput) (*Liability, error) {
	if err := ValidateName(in.Name); err != nil {
		return nil, err
	}
	if err := ValidateType(in.Type); err != nil {
		return nil, err
	}
	if err := ValidateAmount(in.PrincipalAmount); err != nil {
		return nil, err
	}
	if err := ValidateAmount(in.CurrentBalance); err != nil {
		return nil, err
	}

	status := StatusActive
	if in.CurrentBalance.IsZero() {
		status = StatusSettled
	}

	l := &Liability{
		UserID: userID, Name: in.Name, Type: in.Type, PrincipalAmount: in.PrincipalAmount,
		CurrentBalance: in.CurrentBalance, ReferenceDate: in.ReferenceDate, Notes: in.Notes, Status: status,
	}
	if err := s.repo.Create(ctx, l); err != nil {
		return nil, fmt.Errorf("criar passivo: %w", err)
	}
	s.audit.Record(ctx, &userID, "liability.created", "liability", l.ID, nil)
	return l, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (*Liability, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *Service) List(ctx context.Context, userID string, includeSettled bool) ([]*Liability, error) {
	return s.repo.List(ctx, userID, includeSettled)
}

type UpdateInput struct {
	Name   string
	Type   string
	Notes  *string
	Status string
}

func (s *Service) Update(ctx context.Context, userID, id string, in UpdateInput) (*Liability, error) {
	if err := ValidateName(in.Name); err != nil {
		return nil, err
	}
	if err := ValidateType(in.Type); err != nil {
		return nil, err
	}
	if in.Status != StatusActive && in.Status != StatusSettled {
		return nil, fmt.Errorf("status inválido: %s", in.Status)
	}
	if err := s.repo.UpdateDetails(ctx, id, userID, in.Name, in.Type, in.Notes, in.Status); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "liability.updated", "liability", id, nil)
	return s.repo.GetByID(ctx, id, userID)
}

// AddValuation registra uma baixa/atualização do saldo devedor. Quando o
// saldo informado chega a zero, o passivo é automaticamente marcado como
// "settled" (liquidado), preservando o histórico de baixas.
func (s *Service) AddValuation(ctx context.Context, userID, liabilityID string, balance decimal.Decimal, date time.Time, notes *string) (*Valuation, error) {
	if err := ValidateAmount(balance); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetByID(ctx, liabilityID, userID); err != nil {
		return nil, err
	}
	v, err := s.repo.AddValuation(ctx, liabilityID, userID, balance, date, notes)
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "liability.balance_updated", "liability", liabilityID, map[string]any{"balance": balance.String(), "date": date.Format("2006-01-02")})
	return v, nil
}

func (s *Service) ListValuations(ctx context.Context, userID, liabilityID string) ([]*Valuation, error) {
	if _, err := s.repo.GetByID(ctx, liabilityID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListValuations(ctx, liabilityID, userID)
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	s.audit.Record(ctx, &userID, "liability.deleted", "liability", id, nil)
	return nil
}
