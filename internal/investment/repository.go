package investment

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

const investmentColumns = `id, user_id, name, asset_type, institution, ticker, quantity, average_price, invested_amount, market_value, position_date, acquisition_date, maturity_date, include_in_net_worth, notes, status, created_at, updated_at`

// Create insere o investimento e já registra seu valor de mercado inicial
// no histórico de avaliações, de forma atômica.
func (r *Repository) Create(ctx context.Context, inv *Investment) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback()

	const insertInvestment = `
		INSERT INTO investments (user_id, name, asset_type, institution, ticker, quantity, average_price, invested_amount, market_value, position_date, acquisition_date, maturity_date, include_in_net_worth, notes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`
	if err := tx.QueryRowContext(ctx, insertInvestment, inv.UserID, inv.Name, inv.AssetType, inv.Institution, inv.Ticker, inv.Quantity, inv.AveragePrice,
		inv.InvestedAmount, inv.MarketValue, inv.PositionDate, inv.AcquisitionDate, inv.MaturityDate, inv.IncludeInNetWorth, inv.Notes, inv.Status).
		Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		return fmt.Errorf("criar investimento: %w", err)
	}

	const insertValuation = `
		INSERT INTO investment_valuations (investment_id, user_id, market_value, position_date)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.ExecContext(ctx, insertValuation, inv.ID, inv.UserID, inv.MarketValue, inv.PositionDate); err != nil {
		return fmt.Errorf("registrar valor de mercado inicial: %w", err)
	}

	return tx.Commit()
}

func (r *Repository) GetByID(ctx context.Context, id, userID string) (*Investment, error) {
	q := fmt.Sprintf(`SELECT %s FROM investments WHERE id = $1 AND user_id = $2`, investmentColumns)
	return scanInvestment(r.db.QueryRowContext(ctx, q, id, userID))
}

func scanInvestment(row *sql.Row) (*Investment, error) {
	var inv Investment
	err := row.Scan(&inv.ID, &inv.UserID, &inv.Name, &inv.AssetType, &inv.Institution, &inv.Ticker, &inv.Quantity, &inv.AveragePrice,
		&inv.InvestedAmount, &inv.MarketValue, &inv.PositionDate, &inv.AcquisitionDate, &inv.MaturityDate, &inv.IncludeInNetWorth, &inv.Notes, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("consultar investimento: %w", err)
	}
	return &inv, nil
}

func (r *Repository) List(ctx context.Context, userID string, includeClosed bool) ([]*Investment, error) {
	q := fmt.Sprintf(`SELECT %s FROM investments WHERE user_id = $1`, investmentColumns)
	if !includeClosed {
		q += ` AND status = 'active'`
	}
	q += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("listar investimentos: %w", err)
	}
	defer rows.Close()

	var out []*Investment
	for rows.Next() {
		var inv Investment
		if err := rows.Scan(&inv.ID, &inv.UserID, &inv.Name, &inv.AssetType, &inv.Institution, &inv.Ticker, &inv.Quantity, &inv.AveragePrice,
			&inv.InvestedAmount, &inv.MarketValue, &inv.PositionDate, &inv.AcquisitionDate, &inv.MaturityDate, &inv.IncludeInNetWorth, &inv.Notes, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ler investimento: %w", err)
		}
		out = append(out, &inv)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateDetails(ctx context.Context, id, userID string, inv *Investment) error {
	const q = `
		UPDATE investments
		SET name = $3, asset_type = $4, institution = $5, ticker = $6, acquisition_date = $7,
			maturity_date = $8, include_in_net_worth = $9, notes = $10, status = $11, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`
	res, err := r.db.ExecContext(ctx, q, id, userID, inv.Name, inv.AssetType, inv.Institution, inv.Ticker,
		inv.AcquisitionDate, inv.MaturityDate, inv.IncludeInNetWorth, inv.Notes, inv.Status)
	if err != nil {
		return fmt.Errorf("atualizar investimento: %w", err)
	}
	return checkAffected(res)
}

func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM investments WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("excluir investimento: %w", err)
	}
	return checkAffected(res)
}

// HasMovements indica se existem movimentações registradas para o
// investimento, usado para decidir entre excluir ou apenas encerrar (closed).
func (r *Repository) HasMovements(ctx context.Context, id, userID string) (bool, error) {
	var exists bool
	const q = `SELECT EXISTS (SELECT 1 FROM investment_transactions WHERE investment_id = $1 AND user_id = $2)`
	if err := r.db.QueryRowContext(ctx, q, id, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("verificar movimentações: %w", err)
	}
	return exists, nil
}

func (r *Repository) Close(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE investments SET status = 'closed', updated_at = now() WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("encerrar investimento: %w", err)
	}
	return checkAffected(res)
}

// RecordMovement grava a movimentação, atualiza a posição (quantidade,
// preço médio, valor investido) e, quando há conta vinculada, registra a
// perna correspondente em transactions (transfer_in/out) — tudo em uma
// única transação SQL atômica.
func (r *Repository) RecordMovement(ctx context.Context, m *Movement, newQuantity, newAveragePrice *decimal.Decimal, newInvestedAmount, newMarketValue decimal.Decimal) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback()

	const insertMovement = `
		INSERT INTO investment_transactions (user_id, investment_id, account_id, type, amount, quantity, fees, realized_gain, occurred_on, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`
	if err := tx.QueryRowContext(ctx, insertMovement, m.UserID, m.InvestmentID, m.AccountID, m.Type, m.Amount, m.Quantity, m.Fees, m.RealizedGain, m.OccurredOn, m.Notes).
		Scan(&m.ID, &m.CreatedAt); err != nil {
		return fmt.Errorf("registrar movimentação: %w", err)
	}

	const updateInvestment = `
		UPDATE investments SET quantity = $3, average_price = $4, invested_amount = $5, market_value = $6, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`
	if _, err := tx.ExecContext(ctx, updateInvestment, m.InvestmentID, m.UserID, newQuantity, newAveragePrice, newInvestedAmount, newMarketValue); err != nil {
		return fmt.Errorf("atualizar posição: %w", err)
	}

	if m.AccountID != nil {
		legType := "transfer_out"
		description := "Aporte em investimento"
		switch m.Type {
		case TxContribution, TxFee:
			legType = "transfer_out"
			description = movementDescription(m.Type)
		case TxRedemption, TxIncome:
			legType = "transfer_in"
			description = movementDescription(m.Type)
		}
		const insertLeg = `
			INSERT INTO transactions (user_id, account_id, investment_transaction_id, type, status, amount, description, occurred_on)
			VALUES ($1, $2, $3, $4, 'paid', $5, $6, $7)
		`
		if _, err := tx.ExecContext(ctx, insertLeg, m.UserID, *m.AccountID, m.ID, legType, m.Amount, description, m.OccurredOn); err != nil {
			return fmt.Errorf("registrar perna na conta financeira: %w", err)
		}
	}

	return tx.Commit()
}

func movementDescription(t string) string {
	switch t {
	case TxContribution:
		return "Aporte em investimento"
	case TxRedemption:
		return "Resgate de investimento"
	case TxIncome:
		return "Rendimento de investimento"
	case TxFee:
		return "Taxa de investimento"
	default:
		return "Movimentação de investimento"
	}
}

func (r *Repository) ListMovements(ctx context.Context, investmentID, userID string) ([]*Movement, error) {
	const q = `
		SELECT id, user_id, investment_id, account_id, type, amount, quantity, fees, realized_gain, occurred_on, notes, created_at
		FROM investment_transactions
		WHERE investment_id = $1 AND user_id = $2
		ORDER BY occurred_on DESC, created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q, investmentID, userID)
	if err != nil {
		return nil, fmt.Errorf("listar movimentações: %w", err)
	}
	defer rows.Close()

	var out []*Movement
	for rows.Next() {
		var m Movement
		if err := rows.Scan(&m.ID, &m.UserID, &m.InvestmentID, &m.AccountID, &m.Type, &m.Amount, &m.Quantity, &m.Fees, &m.RealizedGain, &m.OccurredOn, &m.Notes, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("ler movimentação: %w", err)
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// AddValuation registra um novo valor de mercado para o investimento,
// preservando o histórico e atualizando a posição atual, de forma atômica.
func (r *Repository) AddValuation(ctx context.Context, investmentID, userID string, marketValue decimal.Decimal, date time.Time, notes *string) (*Valuation, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback()

	v := &Valuation{InvestmentID: investmentID, MarketValue: marketValue, PositionDate: date, Notes: notes}
	const insertValuation = `
		INSERT INTO investment_valuations (investment_id, user_id, market_value, position_date, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	if err := tx.QueryRowContext(ctx, insertValuation, investmentID, userID, marketValue, date, notes).Scan(&v.ID, &v.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrValuationExists
		}
		return nil, fmt.Errorf("registrar valor de mercado: %w", err)
	}

	const updateInvestment = `
		UPDATE investments SET market_value = $3, position_date = $4, updated_at = now()
		WHERE id = $1 AND user_id = $2 AND position_date <= $4
	`
	if _, err := tx.ExecContext(ctx, updateInvestment, investmentID, userID, marketValue, date); err != nil {
		return nil, fmt.Errorf("atualizar valor de mercado atual: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmar valor de mercado: %w", err)
	}
	return v, nil
}

func (r *Repository) ListValuations(ctx context.Context, investmentID, userID string) ([]*Valuation, error) {
	const q = `
		SELECT id, investment_id, market_value, position_date, notes, created_at
		FROM investment_valuations WHERE investment_id = $1 AND user_id = $2
		ORDER BY position_date DESC
	`
	rows, err := r.db.QueryContext(ctx, q, investmentID, userID)
	if err != nil {
		return nil, fmt.Errorf("listar valores de mercado: %w", err)
	}
	defer rows.Close()

	var out []*Valuation
	for rows.Next() {
		var v Valuation
		if err := rows.Scan(&v.ID, &v.InvestmentID, &v.MarketValue, &v.PositionDate, &v.Notes, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("ler valor de mercado: %w", err)
		}
		out = append(out, &v)
	}
	return out, rows.Err()
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
