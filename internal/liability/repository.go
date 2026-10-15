package liability

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create insere o passivo e já registra seu saldo inicial no histórico de
// baixas, de forma atômica, para que o histórico sempre reflita todos os
// saldos conhecidos desde a criação.
func (r *Repository) Create(ctx context.Context, l *Liability) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback()

	const insertLiability = `
		INSERT INTO liabilities (user_id, name, type, principal_amount, current_balance, reference_date, notes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	if err := tx.QueryRowContext(ctx, insertLiability, l.UserID, l.Name, l.Type, l.PrincipalAmount, l.CurrentBalance, l.ReferenceDate, l.Notes, l.Status).
		Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt); err != nil {
		return fmt.Errorf("criar passivo: %w", err)
	}

	const insertValuation = `
		INSERT INTO liability_valuations (liability_id, user_id, balance, reference_date, notes)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := tx.ExecContext(ctx, insertValuation, l.ID, l.UserID, l.CurrentBalance, l.ReferenceDate, l.Notes); err != nil {
		return fmt.Errorf("registrar saldo inicial: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) GetByID(ctx context.Context, id, userID string) (*Liability, error) {
	const q = `
		SELECT id, user_id, name, type, principal_amount, current_balance, reference_date, notes, status, created_at, updated_at
		FROM liabilities WHERE id = $1 AND user_id = $2
	`
	return scanOne(r.db.QueryRowContext(ctx, q, id, userID))
}

func scanOne(row *sql.Row) (*Liability, error) {
	var l Liability
	err := row.Scan(&l.ID, &l.UserID, &l.Name, &l.Type, &l.PrincipalAmount, &l.CurrentBalance, &l.ReferenceDate, &l.Notes, &l.Status, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("consultar passivo: %w", err)
	}
	return &l, nil
}

func (r *Repository) List(ctx context.Context, userID string, includeSettled bool) ([]*Liability, error) {
	q := `
		SELECT id, user_id, name, type, principal_amount, current_balance, reference_date, notes, status, created_at, updated_at
		FROM liabilities WHERE user_id = $1
	`
	if !includeSettled {
		q += ` AND status = 'active'`
	}
	q += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("listar passivos: %w", err)
	}
	defer rows.Close()

	var out []*Liability
	for rows.Next() {
		var l Liability
		if err := rows.Scan(&l.ID, &l.UserID, &l.Name, &l.Type, &l.PrincipalAmount, &l.CurrentBalance, &l.ReferenceDate, &l.Notes, &l.Status, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ler passivo: %w", err)
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateDetails(ctx context.Context, id, userID, name, liabilityType string, notes *string, status string) error {
	const q = `
		UPDATE liabilities SET name = $3, type = $4, notes = $5, status = $6, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`
	res, err := r.db.ExecContext(ctx, q, id, userID, name, liabilityType, notes, status)
	if err != nil {
		return fmt.Errorf("atualizar passivo: %w", err)
	}
	return checkAffected(res)
}

// AddValuation registra uma baixa/atualização de saldo devedor, preservando
// o histórico anterior, e atualiza o saldo atual do passivo de forma
// atômica.
func (r *Repository) AddValuation(ctx context.Context, liabilityID, userID string, balance decimal.Decimal, date time.Time, notes *string) (*Valuation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback()

	v := &Valuation{LiabilityID: liabilityID, Balance: balance, ReferenceDate: date, Notes: notes}
	const insertValuation = `
		INSERT INTO liability_valuations (liability_id, user_id, balance, reference_date, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	if err := tx.QueryRowContext(ctx, insertValuation, liabilityID, userID, balance, date, notes).Scan(&v.ID, &v.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrValuationExists
		}
		return nil, fmt.Errorf("registrar baixa de saldo: %w", err)
	}

	status := StatusActive
	if balance.IsZero() {
		status = StatusSettled
	}
	const updateLiability = `
		UPDATE liabilities SET current_balance = $3, reference_date = $4, status = $5, updated_at = now()
		WHERE id = $1 AND user_id = $2 AND reference_date <= $4
	`
	if _, err := tx.ExecContext(ctx, updateLiability, liabilityID, userID, balance, date, status); err != nil {
		return nil, fmt.Errorf("atualizar saldo atual do passivo: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmar baixa de saldo: %w", err)
	}
	return v, nil
}

func (r *Repository) ListValuations(ctx context.Context, liabilityID, userID string) ([]*Valuation, error) {
	const q = `
		SELECT lv.id, lv.liability_id, lv.balance, lv.reference_date, lv.notes, lv.created_at
		FROM liability_valuations lv
		JOIN liabilities l ON l.id = lv.liability_id
		WHERE lv.liability_id = $1 AND l.user_id = $2
		ORDER BY lv.reference_date DESC
	`
	rows, err := r.db.QueryContext(ctx, q, liabilityID, userID)
	if err != nil {
		return nil, fmt.Errorf("listar baixas de saldo: %w", err)
	}
	defer rows.Close()

	var out []*Valuation
	for rows.Next() {
		var v Valuation
		if err := rows.Scan(&v.ID, &v.LiabilityID, &v.Balance, &v.ReferenceDate, &v.Notes, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("ler baixa de saldo: %w", err)
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM liabilities WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("excluir passivo: %w", err)
	}
	return checkAffected(res)
}

func checkAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar linhas afetadas: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
