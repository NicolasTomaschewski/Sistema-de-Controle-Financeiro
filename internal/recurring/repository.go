package recurring

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const columns = `id, user_id, account_id, category_id, description, planned_amount, periodicity, start_date, end_date, due_day, due_month, active, created_at, updated_at`

func (r *Repository) Create(ctx context.Context, rule *Rule) error {
	const q = `
		INSERT INTO recurring_rules (user_id, account_id, category_id, description, planned_amount, periodicity, start_date, end_date, due_day, due_month, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, q, rule.UserID, rule.AccountID, rule.CategoryID, rule.Description, rule.PlannedAmount,
		rule.Periodicity, rule.StartDate, rule.EndDate, rule.DueDay, rule.DueMonth, rule.Active).
		Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id, userID string) (*Rule, error) {
	q := fmt.Sprintf(`SELECT %s FROM recurring_rules WHERE id = $1 AND user_id = $2`, columns)
	return scanOne(r.db.QueryRowContext(ctx, q, id, userID))
}

func scanOne(row *sql.Row) (*Rule, error) {
	var rule Rule
	err := row.Scan(&rule.ID, &rule.UserID, &rule.AccountID, &rule.CategoryID, &rule.Description, &rule.PlannedAmount,
		&rule.Periodicity, &rule.StartDate, &rule.EndDate, &rule.DueDay, &rule.DueMonth, &rule.Active, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("consultar regra de recorrência: %w", err)
	}
	return &rule, nil
}

func (r *Repository) List(ctx context.Context, userID string, activeOnly bool) ([]*Rule, error) {
	q := fmt.Sprintf(`SELECT %s FROM recurring_rules WHERE user_id = $1`, columns)
	if activeOnly {
		q += ` AND active = true`
	}
	q += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("listar regras de recorrência: %w", err)
	}
	defer rows.Close()

	var out []*Rule
	for rows.Next() {
		var rule Rule
		if err := rows.Scan(&rule.ID, &rule.UserID, &rule.AccountID, &rule.CategoryID, &rule.Description, &rule.PlannedAmount,
			&rule.Periodicity, &rule.StartDate, &rule.EndDate, &rule.DueDay, &rule.DueMonth, &rule.Active, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ler regra de recorrência: %w", err)
		}
		out = append(out, &rule)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id, userID string, rule *Rule) error {
	const q = `
		UPDATE recurring_rules
		SET account_id = $3, category_id = $4, description = $5, planned_amount = $6,
			periodicity = $7, start_date = $8, end_date = $9, due_day = $10, due_month = $11, active = $12, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`
	res, err := r.db.ExecContext(ctx, q, id, userID, rule.AccountID, rule.CategoryID, rule.Description, rule.PlannedAmount,
		rule.Periodicity, rule.StartDate, rule.EndDate, rule.DueDay, rule.DueMonth, rule.Active)
	if err != nil {
		return fmt.Errorf("atualizar regra de recorrência: %w", err)
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
	res, err := r.db.ExecContext(ctx, `DELETE FROM recurring_rules WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("excluir regra de recorrência: %w", err)
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
