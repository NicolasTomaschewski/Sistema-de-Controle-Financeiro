package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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
	Name               string
	Type               string
	Institution        *string
	Currency           string
	InitialBalance     decimal.Decimal
	InitialBalanceDate time.Time
}

func (s *Service) Create(ctx context.Context, userID string, in CreateInput) (*Account, error) {
	if err := ValidateName(in.Name); err != nil {
		return nil, err
	}
	if err := ValidateType(in.Type); err != nil {
		return nil, err
	}
	currency := in.Currency
	if currency == "" {
		currency = "BRL"
	}
	if err := ValidateCurrency(currency); err != nil {
		return nil, err
	}

	a := &Account{
		UserID:             userID,
		Name:               in.Name,
		Type:               in.Type,
		Institution:        in.Institution,
		Currency:           currency,
		InitialBalance:     in.InitialBalance,
		InitialBalanceDate: in.InitialBalanceDate,
		Status:             StatusActive,
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return nil, fmt.Errorf("criar conta: %w", err)
	}

	s.audit.Record(ctx, &userID, "account.created", "financial_account", a.ID, nil)
	return a, nil
}

func (s *Service) Balance(ctx context.Context, userID, id string) (decimal.Decimal, error) {
	return s.repo.CurrentBalance(ctx, id, userID)
}

func (s *Service) Get(ctx context.Context, userID, id string) (*Account, decimal.Decimal, error) {
	a, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, decimal.Zero, err
	}
	balance, err := s.repo.CurrentBalance(ctx, id, userID)
	if err != nil {
		return nil, decimal.Zero, err
	}
	return a, balance, nil
}

type AccountWithBalance struct {
	Account *Account
	Balance decimal.Decimal
}

func (s *Service) List(ctx context.Context, userID string, includeArchived bool) ([]AccountWithBalance, error) {
	accounts, err := s.repo.List(ctx, userID, includeArchived)
	if err != nil {
		return nil, err
	}
	out := make([]AccountWithBalance, 0, len(accounts))
	for _, a := range accounts {
		balance, err := s.repo.CurrentBalance(ctx, a.ID, userID)
		if err != nil {
			return nil, err
		}
		out = append(out, AccountWithBalance{Account: a, Balance: balance})
	}
	return out, nil
}

type UpdateInput struct {
	Name        string
	Type        string
	Institution *string
	Status      string
}

func (s *Service) Update(ctx context.Context, userID, id string, in UpdateInput) (*Account, error) {
	if err := ValidateName(in.Name); err != nil {
		return nil, err
	}
	if err := ValidateType(in.Type); err != nil {
		return nil, err
	}
	if in.Status != StatusActive && in.Status != StatusArchived {
		return nil, fmt.Errorf("status inválido: %s", in.Status)
	}

	if err := s.repo.Update(ctx, id, userID, in.Name, in.Type, in.Institution, in.Status); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "account.updated", "financial_account", id, nil)
	return s.repo.GetByID(ctx, id, userID)
}

func (s *Service) UpdateInitialBalance(ctx context.Context, userID, id string, balance decimal.Decimal, date time.Time) (*Account, error) {
	if err := s.repo.UpdateInitialBalance(ctx, id, userID, balance, date); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "account.initial_balance_changed", "financial_account", id, map[string]any{
		"new_initial_balance": balance.String(),
	})
	return s.repo.GetByID(ctx, id, userID)
}

// Delete exclui a conta somente se não houver movimentações vinculadas;
// caso contrário, arquiva a conta para preservar o histórico financeiro.
func (s *Service) Delete(ctx context.Context, userID, id string) (archived bool, err error) {
	hasTx, err := s.repo.HasTransactions(ctx, id, userID)
	if err != nil {
		return false, err
	}
	if hasTx {
		if err := s.repo.Archive(ctx, id, userID); err != nil {
			return false, err
		}
		s.audit.Record(ctx, &userID, "account.archived", "financial_account", id, nil)
		return true, nil
	}

	if err := s.repo.Delete(ctx, id, userID); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			if archErr := s.repo.Archive(ctx, id, userID); archErr == nil {
				s.audit.Record(ctx, &userID, "account.archived", "financial_account", id, nil)
				return true, nil
			}
		}
		return false, err
	}

	s.audit.Record(ctx, &userID, "account.deleted", "financial_account", id, nil)
	return false, nil
}
