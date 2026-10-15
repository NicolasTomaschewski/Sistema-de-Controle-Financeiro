package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const columns = `id, user_id, account_id, category_id, transfer_id, recurring_rule_id, investment_transaction_id, type, status, amount, description, occurred_on, notes, created_at, updated_at`

func (r *Repository) Create(ctx context.Context, t *Transaction) error {
	const q = `
		INSERT INTO transactions (user_id, account_id, category_id, recurring_rule_id, type, status, amount, description, occurred_on, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, q, t.UserID, t.AccountID, t.CategoryID, t.RecurringRuleID, t.Type, t.Status, t.Amount, t.Description, t.OccurredOn, t.Notes).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// CreateOccurrence materializa a ocorrência de uma regra de recorrência para
// um período específico. É idempotente: se a ocorrência já existir para o
// mesmo (recurring_rule_id, occurred_on), retorna a linha existente em vez
// de duplicá-la (restrição única parcial garante isso também no banco).
func (r *Repository) CreateOccurrence(ctx context.Context, t *Transaction) (created bool, err error) {
	const q = `
		INSERT INTO transactions (user_id, account_id, category_id, recurring_rule_id, type, status, amount, description, occurred_on, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (recurring_rule_id, occurred_on) WHERE recurring_rule_id IS NOT NULL DO NOTHING
		RETURNING id, created_at, updated_at
	`
	err = r.db.QueryRowContext(ctx, q, t.UserID, t.AccountID, t.CategoryID, t.RecurringRuleID, t.Type, t.Status, t.Amount, t.Description, t.OccurredOn, t.Notes).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		existing, getErr := r.GetByRecurringOccurrence(ctx, *t.RecurringRuleID, t.OccurredOn)
		if getErr != nil {
			return false, getErr
		}
		*t = *existing
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("gerar ocorrência de recorrência: %w", err)
	}
	return true, nil
}

func (r *Repository) GetByRecurringOccurrence(ctx context.Context, recurringRuleID string, occurredOn time.Time) (*Transaction, error) {
	q := fmt.Sprintf(`SELECT %s FROM transactions WHERE recurring_rule_id = $1 AND occurred_on = $2`, columns)
	row := r.db.QueryRowContext(ctx, q, recurringRuleID, occurredOn)
	return scanOne(row)
}

func (r *Repository) GetByID(ctx context.Context, id, userID string) (*Transaction, error) {
	q := fmt.Sprintf(`SELECT %s FROM transactions WHERE id = $1 AND user_id = $2`, columns)
	row := r.db.QueryRowContext(ctx, q, id, userID)
	return scanOne(row)
}

func scanOne(row *sql.Row) (*Transaction, error) {
	var t Transaction
	err := row.Scan(&t.ID, &t.UserID, &t.AccountID, &t.CategoryID, &t.TransferID, &t.RecurringRuleID, &t.InvestmentTransactionID, &t.Type, &t.Status, &t.Amount, &t.Description, &t.OccurredOn, &t.Notes, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("consultar lançamento: %w", err)
	}
	return &t, nil
}

type Filter struct {
	AccountID  string
	CategoryID string
	Type       string
	Status     string
	From       *time.Time
	To         *time.Time
	Limit      int
	Offset     int
}

// List retorna os lançamentos do usuário autenticado que atendem aos
// filtros informados, sempre restritos ao próprio usuário na cláusula WHERE.
func (r *Repository) List(ctx context.Context, userID string, f Filter) ([]*Transaction, int, error) {
	where := []string{"user_id = $1"}
	args := []any{userID}

	addFilter := func(cond string, val any) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}

	if f.AccountID != "" {
		addFilter("account_id = $%d", f.AccountID)
	}
	if f.CategoryID != "" {
		addFilter("category_id = $%d", f.CategoryID)
	}
	if f.Type != "" {
		addFilter("type = $%d", f.Type)
	}
	if f.Status != "" {
		addFilter("status = $%d", f.Status)
	}
	if f.From != nil {
		addFilter("occurred_on >= $%d", *f.From)
	}
	if f.To != nil {
		addFilter("occurred_on <= $%d", *f.To)
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQ := fmt.Sprintf(`SELECT count(*) FROM transactions WHERE %s`, whereClause)
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("contar lançamentos: %w", err)
	}

	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	q := fmt.Sprintf(
		`SELECT %s FROM transactions WHERE %s ORDER BY occurred_on DESC, created_at DESC LIMIT $%d OFFSET $%d`,
		columns, whereClause, len(args)-1, len(args),
	)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listar lançamentos: %w", err)
	}
	defer rows.Close()

	var out []*Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.AccountID, &t.CategoryID, &t.TransferID, &t.RecurringRuleID, &t.InvestmentTransactionID, &t.Type, &t.Status, &t.Amount, &t.Description, &t.OccurredOn, &t.Notes, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("ler lançamento: %w", err)
		}
		out = append(out, &t)
	}
	return out, total, rows.Err()
}

type UpdateFields struct {
	AccountID   string
	CategoryID  *string
	Status      string
	Amount      decimal.Decimal
	Description string
	OccurredOn  time.Time
	Notes       *string
}

func (r *Repository) Update(ctx context.Context, id, userID string, f UpdateFields) error {
	const q = `
		UPDATE transactions
		SET account_id = $3, category_id = $4, status = $5, amount = $6, description = $7, occurred_on = $8, notes = $9, updated_at = now()
		WHERE id = $1 AND user_id = $2 AND transfer_id IS NULL
	`
	res, err := r.db.ExecContext(ctx, q, id, userID, f.AccountID, f.CategoryID, f.Status, f.Amount, f.Description, f.OccurredOn, f.Notes)
	if err != nil {
		return fmt.Errorf("atualizar lançamento: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar linhas afetadas: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM transactions WHERE id = $1 AND user_id = $2 AND transfer_id IS NULL`, id, userID)
	if err != nil {
		return fmt.Errorf("excluir lançamento: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar linhas afetadas: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

type MonthlySummary struct {
	TotalIncome  decimal.Decimal
	TotalExpense decimal.Decimal
	Result       decimal.Decimal
}

// Summary calcula totais de receitas e despesas efetivamente pagas/recebidas
// no período informado. Transferências nunca entram neste cálculo.
func (r *Repository) Summary(ctx context.Context, userID string, from, to time.Time) (MonthlySummary, error) {
	const q = `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)
		FROM transactions
		WHERE user_id = $1 AND status = 'paid' AND occurred_on BETWEEN $2 AND $3
			AND type IN ('income', 'expense')
	`
	var s MonthlySummary
	err := r.db.QueryRowContext(ctx, q, userID, from, to).Scan(&s.TotalIncome, &s.TotalExpense)
	if err != nil {
		return MonthlySummary{}, fmt.Errorf("calcular resumo do período: %w", err)
	}
	s.Result = s.TotalIncome.Sub(s.TotalExpense)
	return s, nil
}

type CategoryBreakdownItem struct {
	CategoryID   *string
	CategoryName string
	Total        decimal.Decimal
}

// ExpensesByCategory agrupa as despesas pagas no período por categoria.
func (r *Repository) ExpensesByCategory(ctx context.Context, userID string, from, to time.Time) ([]CategoryBreakdownItem, error) {
	return r.byCategory(ctx, userID, TypeExpense, from, to)
}

// IncomeByCategory agrupa as receitas recebidas no período por categoria (fonte de renda).
func (r *Repository) IncomeByCategory(ctx context.Context, userID string, from, to time.Time) ([]CategoryBreakdownItem, error) {
	return r.byCategory(ctx, userID, TypeIncome, from, to)
}

func (r *Repository) byCategory(ctx context.Context, userID, txType string, from, to time.Time) ([]CategoryBreakdownItem, error) {
	const q = `
		SELECT t.category_id, COALESCE(c.name, 'Sem categoria'), SUM(t.amount)
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.user_id = $1 AND t.status = 'paid' AND t.type = $2
			AND t.occurred_on BETWEEN $3 AND $4
		GROUP BY t.category_id, c.name
		ORDER BY SUM(t.amount) DESC
	`
	rows, err := r.db.QueryContext(ctx, q, userID, txType, from, to)
	if err != nil {
		return nil, fmt.Errorf("agrupar despesas por categoria: %w", err)
	}
	defer rows.Close()

	var out []CategoryBreakdownItem
	for rows.Next() {
		var item CategoryBreakdownItem
		if err := rows.Scan(&item.CategoryID, &item.CategoryName, &item.Total); err != nil {
			return nil, fmt.Errorf("ler agrupamento por categoria: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
