package asset

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

// Create insere o ativo e já registra sua avaliação inicial no histórico,
// de forma atômica, para que o histórico de avaliações sempre reflita
// todos os valores conhecidos do ativo desde a criação.
func (r *Repository) Create(ctx context.Context, a *Asset) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback()

	const insertAsset = `
		INSERT INTO assets (user_id, name, type, current_value, valuation_date, notes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	if err := tx.QueryRowContext(ctx, insertAsset, a.UserID, a.Name, a.Type, a.CurrentValue, a.ValuationDate, a.Notes, a.Status).
		Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return fmt.Errorf("criar ativo: %w", err)
	}

	const insertValuation = `
		INSERT INTO asset_valuations (asset_id, user_id, value, valuation_date, notes)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := tx.ExecContext(ctx, insertValuation, a.ID, a.UserID, a.CurrentValue, a.ValuationDate, a.Notes); err != nil {
		return fmt.Errorf("registrar avaliação inicial: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) GetByID(ctx context.Context, id, userID string) (*Asset, error) {
	const q = `
		SELECT id, user_id, name, type, current_value, valuation_date, notes, status, created_at, updated_at
		FROM assets WHERE id = $1 AND user_id = $2
	`
	return scanOne(r.db.QueryRowContext(ctx, q, id, userID))
}

func scanOne(row *sql.Row) (*Asset, error) {
	var a Asset
	err := row.Scan(&a.ID, &a.UserID, &a.Name, &a.Type, &a.CurrentValue, &a.ValuationDate, &a.Notes, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("consultar ativo: %w", err)
	}
	return &a, nil
}

func (r *Repository) List(ctx context.Context, userID string, includeArchived bool) ([]*Asset, error) {
	q := `
		SELECT id, user_id, name, type, current_value, valuation_date, notes, status, created_at, updated_at
		FROM assets WHERE user_id = $1
	`
	if !includeArchived {
		q += ` AND status = 'active'`
	}
	q += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("listar ativos: %w", err)
	}
	defer rows.Close()

	var out []*Asset
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Type, &a.CurrentValue, &a.ValuationDate, &a.Notes, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ler ativo: %w", err)
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateDetails(ctx context.Context, id, userID, name, assetType string, notes *string, status string) error {
	const q = `
		UPDATE assets SET name = $3, type = $4, notes = $5, status = $6, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`
	res, err := r.db.ExecContext(ctx, q, id, userID, name, assetType, notes, status)
	if err != nil {
		return fmt.Errorf("atualizar ativo: %w", err)
	}
	return checkAffected(res)
}

// AddValuation registra uma nova avaliação para o ativo e atualiza seu
// valor atual/data de referência, de forma atômica. O histórico de
// avaliações anteriores nunca é sobrescrito ou removido.
func (r *Repository) AddValuation(ctx context.Context, assetID, userID string, value decimal.Decimal, date time.Time, notes *string) (*Valuation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback()

	v := &Valuation{AssetID: assetID, Value: value, ValuationDate: date, Notes: notes}
	const insertValuation = `
		INSERT INTO asset_valuations (asset_id, user_id, value, valuation_date, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	if err := tx.QueryRowContext(ctx, insertValuation, assetID, userID, value, date, notes).Scan(&v.ID, &v.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrValuationExists
		}
		return nil, fmt.Errorf("registrar avaliação: %w", err)
	}

	const updateAsset = `
		UPDATE assets SET current_value = $3, valuation_date = $4, updated_at = now()
		WHERE id = $1 AND user_id = $2 AND valuation_date <= $4
	`
	if _, err := tx.ExecContext(ctx, updateAsset, assetID, userID, value, date); err != nil {
		return nil, fmt.Errorf("atualizar valor atual do ativo: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmar avaliação: %w", err)
	}
	return v, nil
}

// ListValuations retorna o histórico de avaliações do ativo, do mais
// recente para o mais antigo.
func (r *Repository) ListValuations(ctx context.Context, assetID, userID string) ([]*Valuation, error) {
	const q = `
		SELECT av.id, av.asset_id, av.value, av.valuation_date, av.notes, av.created_at
		FROM asset_valuations av
		JOIN assets a ON a.id = av.asset_id
		WHERE av.asset_id = $1 AND a.user_id = $2
		ORDER BY av.valuation_date DESC
	`
	rows, err := r.db.QueryContext(ctx, q, assetID, userID)
	if err != nil {
		return nil, fmt.Errorf("listar avaliações: %w", err)
	}
	defer rows.Close()

	var out []*Valuation
	for rows.Next() {
		var v Valuation
		if err := rows.Scan(&v.ID, &v.AssetID, &v.Value, &v.ValuationDate, &v.Notes, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("ler avaliação: %w", err)
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}

func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM assets WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("excluir ativo: %w", err)
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
