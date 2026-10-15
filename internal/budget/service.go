package budget

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/nicolastomaschewski/controle-financeiro/internal/audit"
	"github.com/nicolastomaschewski/controle-financeiro/internal/category"
	"github.com/nicolastomaschewski/controle-financeiro/internal/transaction"
)

type Service struct {
	repo       *Repository
	categories *category.Repository
	audit      *audit.Recorder
}

func NewService(repo *Repository, categories *category.Repository, audit *audit.Recorder) *Service {
	return &Service{repo: repo, categories: categories, audit: audit}
}

type Input struct {
	CategoryID    string
	Year          int
	Month         int
	PlannedAmount decimal.Decimal
}

func (s *Service) validateCategory(ctx context.Context, userID, categoryID string) error {
	cat, err := s.categories.GetVisible(ctx, categoryID, userID)
	if err != nil {
		if errors.Is(err, category.ErrNotFound) {
			return transaction.ErrCategoryNotVisible
		}
		return fmt.Errorf("verificar categoria: %w", err)
	}
	if cat.Kind != category.KindExpense {
		return transaction.ErrCategoryKindMismatch
	}
	return nil
}

func (s *Service) Create(ctx context.Context, userID string, in Input) (*Budget, error) {
	if err := ValidateAmount(in.PlannedAmount); err != nil {
		return nil, err
	}
	if err := ValidatePeriod(in.Year, in.Month); err != nil {
		return nil, err
	}
	if err := s.validateCategory(ctx, userID, in.CategoryID); err != nil {
		return nil, err
	}

	b := &Budget{UserID: userID, CategoryID: in.CategoryID, Year: in.Year, Month: in.Month, PlannedAmount: in.PlannedAmount}
	if err := s.repo.Create(ctx, b); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "budget.created", "budget", b.ID, nil)
	return b, nil
}

// Update altera apenas o valor planejado do orçamento. Isto nunca modifica
// retroativamente os lançamentos já registrados — o gasto realizado é
// sempre recalculado a partir das transações, nunca armazenado no orçamento.
func (s *Service) Update(ctx context.Context, userID, id string, amount decimal.Decimal) (*Budget, error) {
	if err := ValidateAmount(amount); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateAmount(ctx, id, userID, amount); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "budget.updated", "budget", id, nil)
	return s.repo.GetByID(ctx, id, userID)
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	s.audit.Record(ctx, &userID, "budget.deleted", "budget", id, nil)
	return nil
}

// SummaryByPeriod retorna, para cada orçamento definido no período, o valor
// planejado, o gasto realizado, o saldo disponível e o percentual consumido.
func (s *Service) SummaryByPeriod(ctx context.Context, userID string, year, month int) ([]Summary, error) {
	budgets, err := s.repo.ListByPeriod(ctx, userID, year, month)
	if err != nil {
		return nil, err
	}

	out := make([]Summary, 0, len(budgets))
	for _, b := range budgets {
		cat, err := s.categories.GetVisible(ctx, b.CategoryID, userID)
		if err != nil {
			return nil, err
		}
		spent, err := s.repo.SpentByCategory(ctx, userID, b.CategoryID, year, month)
		if err != nil {
			return nil, err
		}

		available := b.PlannedAmount.Sub(spent)
		var percent decimal.Decimal
		if !b.PlannedAmount.IsZero() {
			percent = spent.Div(b.PlannedAmount).Mul(decimal.NewFromInt(100))
		}

		out = append(out, Summary{
			CategoryID: b.CategoryID, CategoryName: cat.Name, PlannedAmount: b.PlannedAmount,
			SpentAmount: spent, AvailableAmount: available, PercentUsed: percent,
			OverLimit: spent.GreaterThan(b.PlannedAmount),
		})
	}
	return out, nil
}
