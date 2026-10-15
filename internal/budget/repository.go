package budget

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, b *Budget) error {
	const q = `
		INSERT INTO budgets (user_id, category_id, year, month, planned_amount)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, q, b.UserID, b.CategoryID, b.Year, b.Month, b.PlannedAmount).
		Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return fmt.Errorf("criar orçamento: %w", err)
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id, userID string) (*Budget, error) {
	const q = `
		SELECT id, user_id, category_id, year, month, planned_amount, created_at, updated_at
		FROM budgets WHERE id = $1 AND user_id = $2
	`
	var b Budget
	err := r.db.QueryRowContext(ctx, q, id, userID).Scan(&b.ID, &b.UserID, &b.CategoryID, &b.Year, &b.Month, &b.PlannedAmount, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("consultar orçamento: %w", err)
	}
	return &b, nil
}

func (r *Repository) ListByPeriod(ctx context.Context, userID string, year, month int) ([]*Budget, error) {
	const q = `
		SELECT id, user_id, category_id, year, month, planned_amount, created_at, updated_at
		FROM budgets WHERE user_id = $1 AND year = $2 AND month = $3
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, q, userID, year, month)
	if err != nil {
		return nil, fmt.Errorf("listar orçamentos: %w", err)
	}
	defer rows.Close()

	var out []*Budget
	for rows.Next() {
		var b Budget
		if err := rows.Scan(&b.ID, &b.UserID, &b.CategoryID, &b.Year, &b.Month, &b.PlannedAmount, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ler orçamento: %w", err)
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateAmount(ctx context.Context, id, userID string, amount decimal.Decimal) error {
	const q = `UPDATE budgets SET planned_amount = $3, updated_at = now() WHERE id = $1 AND user_id = $2`
	res, err := r.db.ExecContext(ctx, q, id, userID, amount)
	if err != nil {
		return fmt.Errorf("atualizar orçamento: %w", err)
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
	res, err := r.db.ExecContext(ctx, `DELETE FROM budgets WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("excluir orçamento: %w", err)
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

// SpentByCategory retorna o total efetivamente pago no período para a
// categoria informada. Não inclui lançamentos pendentes, planejados ou
// cancelados — apenas despesas realizadas contam para o orçamento.
func (r *Repository) SpentByCategory(ctx context.Context, userID, categoryID string, year, month int) (decimal.Decimal, error) {
	const q = `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE user_id = $1 AND category_id = $2 AND type = 'expense' AND status = 'paid'
			AND EXTRACT(YEAR FROM occurred_on) = $3 AND EXTRACT(MONTH FROM occurred_on) = $4
	`
	var spent decimal.Decimal
	if err := r.db.QueryRowContext(ctx, q, userID, categoryID, year, month).Scan(&spent); err != nil {
		return decimal.Zero, fmt.Errorf("calcular gasto por categoria: %w", err)
	}
	return spent, nil
}
