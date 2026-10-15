package recurring

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/nicolastomaschewski/controle-financeiro/internal/account"
	"github.com/nicolastomaschewski/controle-financeiro/internal/audit"
	"github.com/nicolastomaschewski/controle-financeiro/internal/category"
	"github.com/nicolastomaschewski/controle-financeiro/internal/transaction"
)

var ErrUnsupportedPeriodicityForGeneration = errors.New("geração automática só é suportada para periodicidades mensal e anual nesta versão")

type Service struct {
	repo         *Repository
	transactions *transaction.Repository
	accounts     *account.Repository
	categories   *category.Repository
	audit        *audit.Recorder
}

func NewService(repo *Repository, transactions *transaction.Repository, accounts *account.Repository, categories *category.Repository, audit *audit.Recorder) *Service {
	return &Service{repo: repo, transactions: transactions, accounts: accounts, categories: categories, audit: audit}
}

type Input struct {
	AccountID     *string
	CategoryID    *string
	Description   string
	PlannedAmount decimal.Decimal
	Periodicity   string
	StartDate     time.Time
	EndDate       *time.Time
	DueDay        *int
	DueMonth      *int
	Active        bool
}

func (s *Service) validate(ctx context.Context, userID string, in Input) error {
	if err := ValidateDescription(in.Description); err != nil {
		return err
	}
	if err := ValidateAmount(in.PlannedAmount); err != nil {
		return err
	}
	if err := ValidatePeriodicity(in.Periodicity); err != nil {
		return err
	}
	if err := ValidateDueDay(in.DueDay); err != nil {
		return err
	}
	if err := ValidateDueMonth(in.Periodicity, in.DueMonth); err != nil {
		return err
	}
	if in.AccountID != nil {
		if _, err := s.accounts.GetByID(ctx, *in.AccountID, userID); err != nil {
			if errors.Is(err, account.ErrNotFound) {
				return transaction.ErrAccountNotOwned
			}
			return fmt.Errorf("verificar conta: %w", err)
		}
	}
	if in.CategoryID != nil {
		cat, err := s.categories.GetVisible(ctx, *in.CategoryID, userID)
		if err != nil {
			if errors.Is(err, category.ErrNotFound) {
				return transaction.ErrCategoryNotVisible
			}
			return fmt.Errorf("verificar categoria: %w", err)
		}
		if cat.Kind != category.KindExpense {
			return transaction.ErrCategoryKindMismatch
		}
	}
	return nil
}

func (s *Service) Create(ctx context.Context, userID string, in Input) (*Rule, error) {
	if err := s.validate(ctx, userID, in); err != nil {
		return nil, err
	}
	rule := &Rule{
		UserID: userID, AccountID: in.AccountID, CategoryID: in.CategoryID,
		Description: in.Description, PlannedAmount: in.PlannedAmount, Periodicity: in.Periodicity,
		StartDate: in.StartDate, EndDate: in.EndDate, DueDay: in.DueDay, DueMonth: in.DueMonth, Active: true,
	}
	if err := s.repo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("criar regra de recorrência: %w", err)
	}
	s.audit.Record(ctx, &userID, "recurring_rule.created", "recurring_rule", rule.ID, nil)
	return rule, nil
}

func (s *Service) List(ctx context.Context, userID string, activeOnly bool) ([]*Rule, error) {
	return s.repo.List(ctx, userID, activeOnly)
}

func (s *Service) Get(ctx context.Context, userID, id string) (*Rule, error) {
	return s.repo.GetByID(ctx, id, userID)
}

// Update altera a regra de recorrência. Como as ocorrências já geradas são
// lançamentos independentes na tabela de transações, alterar a regra afeta
// apenas as futuras gerações — o histórico já materializado não é tocado.
func (s *Service) Update(ctx context.Context, userID, id string, in Input) (*Rule, error) {
	if _, err := s.repo.GetByID(ctx, id, userID); err != nil {
		return nil, err
	}
	if err := s.validate(ctx, userID, in); err != nil {
		return nil, err
	}

	rule := &Rule{
		AccountID: in.AccountID, CategoryID: in.CategoryID, Description: in.Description,
		PlannedAmount: in.PlannedAmount, Periodicity: in.Periodicity, StartDate: in.StartDate,
		EndDate: in.EndDate, DueDay: in.DueDay, DueMonth: in.DueMonth, Active: in.Active,
	}
	if err := s.repo.Update(ctx, id, userID, rule); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, &userID, "recurring_rule.updated", "recurring_rule", id, nil)
	return s.repo.GetByID(ctx, id, userID)
}

func (s *Service) Delete(ctx context.Context, userID, id string) error {
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	s.audit.Record(ctx, &userID, "recurring_rule.deleted", "recurring_rule", id, nil)
	return nil
}

// Summary consolida o custo de vida fixo do usuário a partir das regras vigentes hoje.
func (s *Service) Summary(ctx context.Context, userID string) (CostSummary, error) {
	rules, err := s.repo.List(ctx, userID, true)
	if err != nil {
		return CostSummary{}, err
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return SummarizeCosts(rules, today), nil
}

// GenerateOccurrence materializa a ocorrência da regra para o ano/mês
// informados (regras anuais apenas no mês de vencimento) como um lançamento identificável (status "pending" se já
// vencida, "planned" se ainda no futuro). É idempotente: se a ocorrência já
// existir para esse período, apenas a retorna, sem duplicar.
func (s *Service) GenerateOccurrence(ctx context.Context, userID, ruleID string, year int, month time.Month) (*transaction.Transaction, bool, error) {
	rule, err := s.repo.GetByID(ctx, ruleID, userID)
	if err != nil {
		return nil, false, err
	}
	if !rule.Active {
		return nil, false, ErrRuleInactive
	}
	if rule.Periodicity != PeriodicityMonthly && rule.Periodicity != PeriodicityYearly {
		return nil, false, ErrUnsupportedPeriodicityForGeneration
	}
	if !rule.IsDueInMonth(month) {
		return nil, false, ErrNotDueInMonth
	}

	dueDate := rule.OccurrenceDate(year, month)
	if !rule.IsWithinValidity(dueDate) {
		return nil, false, ErrOutsideValidity
	}

	status := transaction.StatusPending
	if dueDate.After(time.Now().UTC()) {
		status = transaction.StatusPlanned
	}

	accountID := ""
	if rule.AccountID != nil {
		accountID = *rule.AccountID
	} else if accountID, err = s.accounts.EnsureDefault(ctx, userID); err != nil {
		return nil, false, fmt.Errorf("obter conta padrão: %w", err)
	}

	t := &transaction.Transaction{
		UserID: userID, AccountID: accountID, CategoryID: rule.CategoryID, RecurringRuleID: &rule.ID,
		Type: transaction.TypeExpense, Status: status, Amount: rule.PlannedAmount,
		Description: rule.Description, OccurredOn: dueDate,
	}

	created, err := s.transactions.CreateOccurrence(ctx, t)
	if err != nil {
		return nil, false, fmt.Errorf("materializar ocorrência: %w", err)
	}
	if created {
		s.audit.Record(ctx, &userID, "recurring_rule.occurrence_generated", "transaction", t.ID, map[string]any{"recurring_rule_id": ruleID})
	}
	return t, created, nil
}
