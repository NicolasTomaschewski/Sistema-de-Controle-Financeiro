package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/nicolastomaschewski/controle-financeiro/internal/account"
	"github.com/nicolastomaschewski/controle-financeiro/internal/audit"
	"github.com/nicolastomaschewski/controle-financeiro/internal/category"
)

type Service struct {
	repo       *Repository
	accounts   *account.Repository
	categories *category.Repository
	audit      *audit.Recorder
}

func NewService(repo *Repository, accounts *account.Repository, categories *category.Repository, audit *audit.Recorder) *Service {
	return &Service{repo: repo, accounts: accounts, categories: categories, audit: audit}
}

type Input struct {
	AccountID   string
	CategoryID  *string
	Type        string
	Status      string
	Amount      decimal.Decimal
	Description string
	OccurredOn  time.Time
	Notes       *string
}

// resolveAccount aplica a conta padrão do usuário quando o lançamento não
// informa uma conta: a interface não expõe mais a gestão de contas.
func (s *Service) resolveAccount(ctx context.Context, userID string, in *Input) error {
	if in.AccountID != "" {
		return nil
	}
	id, err := s.accounts.EnsureDefault(ctx, userID)
	if err != nil {
		return err
	}
	in.AccountID = id
	return nil
}

func (s *Service) validateCommon(ctx context.Context, userID string, in Input) error {
	if err := ValidateType(in.Type); err != nil {
		return err
	}
	if err := ValidateStatus(in.Status); err != nil {
		return err
	}
	if err := ValidateAmount(in.Amount); err != nil {
		return err
	}
	if err := ValidateDescription(in.Description); err != nil {
		return err
	}

	if _, err := s.accounts.GetByID(ctx, in.AccountID, userID); err != nil {
		if errors.Is(err, account.ErrNotFound) {
			return ErrAccountNotOwned
		}
		return fmt.Errorf("verificar conta: %w", err)
	}

	if in.CategoryID != nil {
		cat, err := s.categories.GetVisible(ctx, *in.CategoryID, userID)
		if err != nil {
			if errors.Is(err, category.ErrNotFound) {
				return ErrCategoryNotVisible
			}
			return fmt.Errorf("verificar categoria: %w", err)
		}
		if cat.Kind != in.Type {
			return ErrCategoryKindMismatch
		}
	}

	return nil
}

func (s *Service) Create(ctx context.Context, userID string, in Input) (*Transaction, error) {
	if err := s.resolveAccount(ctx, userID, &in); err != nil {
		return nil, err
	}
	if err := s.validateCommon(ctx, userID, in); err != nil {
		return nil, err
	}

	t := &Transaction{
		UserID: userID, AccountID: in.AccountID, CategoryID: in.CategoryID,
		Type: in.Type, Status: in.Status, Amount: in.Amount,
		Description: in.Description, OccurredOn: in.OccurredOn, Notes: in.Notes,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("criar lançamento: %w", err)
	}
	s.audit.Record(ctx, &userID, "transaction.created", "transaction", t.ID, map[string]any{"type": t.Type})
	return t, nil
}

func (s *Service) Get(ctx context.Context, userID, id string) (*Transaction, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *Service) List(ctx context.Context, userID string, f Filter) ([]*Transaction, int, error) {
	return s.repo.List(ctx, userID, f)
}

func (s *Service) Update(ctx context.Context, userID, id string, in Input) (*Transaction, error) {
	existing, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if existing.IsTransferLeg() {
		return nil, ErrIsTransferLeg
	}
	// O tipo (receita/despesa) de um lançamento não é alterável após a
	// criação: trocar entre receita e despesa equivale a criar um novo
	// lançamento, evitando ambiguidade sobre o histórico.
	in.Type = existing.Type
	if in.AccountID == "" {
		in.AccountID = existing.AccountID
	}

	if err := s.validateCommon(ctx, userID, in); err != nil {
		return nil, err
	}

	err = s.repo.Update(ctx, id, userID, UpdateFields{
		AccountID: in.AccountID, CategoryID: in.CategoryID, Status: in.Status,
		Amount: in.Amount, Description: in.Description, OccurredOn: in.OccurredOn, Notes: in.Notes,
	})
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "transaction.updated", "transaction", id, nil)
	return s.repo.GetByID(ctx, id, userID)
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	existing, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return err
	}
	if existing.IsTransferLeg() {
		return ErrIsTransferLeg
	}
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	s.audit.Record(ctx, &userID, "transaction.deleted", "transaction", id, nil)
	return nil
}

type MonthlyReport struct {
	Summary            MonthlySummary
	ExpensesByCategory []CategoryBreakdownItem
	IncomeByCategory   []CategoryBreakdownItem
	PreviousMonth      *MonthlySummary
}

func (s *Service) MonthlyReport(ctx context.Context, userID string, year int, month time.Month, compareWithPreviousMonth bool) (MonthlyReport, error) {
	from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0).Add(-24 * time.Hour)

	summary, err := s.repo.Summary(ctx, userID, from, to)
	if err != nil {
		return MonthlyReport{}, err
	}
	byCategory, err := s.repo.ExpensesByCategory(ctx, userID, from, to)
	if err != nil {
		return MonthlyReport{}, err
	}

	incomeByCategory, err := s.repo.IncomeByCategory(ctx, userID, from, to)
	if err != nil {
		return MonthlyReport{}, err
	}

	report := MonthlyReport{Summary: summary, ExpensesByCategory: byCategory, IncomeByCategory: incomeByCategory}

	if compareWithPreviousMonth {
		prevFrom := from.AddDate(0, -1, 0)
		prevTo := from.Add(-24 * time.Hour)
		prevSummary, err := s.repo.Summary(ctx, userID, prevFrom, prevTo)
		if err != nil {
			return MonthlyReport{}, err
		}
		report.PreviousMonth = &prevSummary
	}

	return report, nil
}

// AnnualReport consolida receitas e despesas efetivadas de 1º de janeiro a
// 31 de dezembro do ano informado, com as mesmas regras do resumo mensal.
func (s *Service) AnnualReport(ctx context.Context, userID string, year int) (MonthlyReport, error) {
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)

	summary, err := s.repo.Summary(ctx, userID, from, to)
	if err != nil {
		return MonthlyReport{}, err
	}
	expenses, err := s.repo.ExpensesByCategory(ctx, userID, from, to)
	if err != nil {
		return MonthlyReport{}, err
	}
	income, err := s.repo.IncomeByCategory(ctx, userID, from, to)
	if err != nil {
		return MonthlyReport{}, err
	}
	return MonthlyReport{Summary: summary, ExpensesByCategory: expenses, IncomeByCategory: income}, nil
}
