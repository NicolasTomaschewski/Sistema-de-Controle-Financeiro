package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, a *Account) error {
	const q = `
		INSERT INTO financial_accounts (user_id, name, type, institution, currency, initial_balance, initial_balance_date, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, q, a.UserID, a.Name, a.Type, a.Institution, a.Currency, a.InitialBalance, a.InitialBalanceDate, a.Status).
		Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// GetByID carrega uma conta garantindo, na própria consulta, que ela
// pertence ao usuário autenticado — nunca confiando apenas na interface.
func (r *Repository) GetByID(ctx context.Context, id, userID string) (*Account, error) {
	const q = `
		SELECT id, user_id, name, type, institution, currency, initial_balance, initial_balance_date, status, created_at, updated_at
		FROM financial_accounts
		WHERE id = $1 AND user_id = $2
	`
	return r.scanOne(r.db.QueryRowContext(ctx, q, id, userID))
}

func (r *Repository) scanOne(row *sql.Row) (*Account, error) {
	var a Account
	err := row.Scan(&a.ID, &a.UserID, &a.Name, &a.Type, &a.Institution, &a.Currency, &a.InitialBalance, &a.InitialBalanceDate, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("consultar conta: %w", err)
	}
	return &a, nil
}

func (r *Repository) List(ctx context.Context, userID string, includeArchived bool) ([]*Account, error) {
	q := `
		SELECT id, user_id, name, type, institution, currency, initial_balance, initial_balance_date, status, created_at, updated_at
		FROM financial_accounts
		WHERE user_id = $1
	`
	if !includeArchived {
		q += ` AND status = 'active'`
	}
	q += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("listar contas: %w", err)
	}
	defer rows.Close()

	var out []*Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Type, &a.Institution, &a.Currency, &a.InitialBalance, &a.InitialBalanceDate, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ler conta: %w", err)
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

// Update altera os campos editáveis da conta, sempre restrito ao
// proprietário autenticado.
func (r *Repository) Update(ctx context.Context, id, userID, name, accType string, institution *string, status string) error {
	const q = `
		UPDATE financial_accounts
		SET name = $3, type = $4, institution = $5, status = $6, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`
	res, err := r.db.ExecContext(ctx, q, id, userID, name, accType, institution, status)
	if err != nil {
		return fmt.Errorf("atualizar conta: %w", err)
	}
	return checkAffected(res)
}

// UpdateInitialBalance altera o saldo inicial e sua data de referência.
// Como o saldo atual é sempre recalculado a partir do saldo inicial e das
// movimentações, esta operação não deixa inconsistências: o novo saldo
// inicial simplesmente passa a valer para todos os cálculos futuros.
func (r *Repository) UpdateInitialBalance(ctx context.Context, id, userID string, balance decimal.Decimal, date time.Time) error {
	const q = `
		UPDATE financial_accounts
		SET initial_balance = $3, initial_balance_date = $4, updated_at = now()
		WHERE id = $1 AND user_id = $2
	`
	res, err := r.db.ExecContext(ctx, q, id, userID, balance, date)
	if err != nil {
		return fmt.Errorf("atualizar saldo inicial: %w", err)
	}
	return checkAffected(res)
}

func (r *Repository) Delete(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM financial_accounts WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("excluir conta: %w", err)
	}
	return checkAffected(res)
}

func (r *Repository) Archive(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE financial_accounts SET status = 'archived', updated_at = now() WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("arquivar conta: %w", err)
	}
	return checkAffected(res)
}

// HasTransactions indica se existe qualquer movimentação vinculada à conta,
// usado para decidir entre excluir ou arquivar.
func (r *Repository) HasTransactions(ctx context.Context, id, userID string) (bool, error) {
	var exists bool
	const q = `SELECT EXISTS (SELECT 1 FROM transactions WHERE account_id = $1 AND user_id = $2)`
	if err := r.db.QueryRowContext(ctx, q, id, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("verificar movimentações da conta: %w", err)
	}
	return exists, nil
}

// CurrentBalance calcula o saldo atual como saldo inicial mais a soma das
// movimentações efetivamente pagas/recebidas (status = 'paid'). Movimentos
// pendentes, planejados ou cancelados não afetam o saldo.
func (r *Repository) CurrentBalance(ctx context.Context, id, userID string) (decimal.Decimal, error) {
	const q = `
		SELECT a.initial_balance + COALESCE(SUM(
			CASE
				WHEN t.type IN ('income', 'transfer_in') THEN t.amount
				WHEN t.type IN ('expense', 'transfer_out') THEN -t.amount
				ELSE 0
			END
		), 0)
		FROM financial_accounts a
		LEFT JOIN transactions t ON t.account_id = a.id AND t.status = 'paid'
		WHERE a.id = $1 AND a.user_id = $2
		GROUP BY a.id, a.initial_balance
	`
	var balance decimal.Decimal
	err := r.db.QueryRowContext(ctx, q, id, userID).Scan(&balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return decimal.Zero, ErrNotFound
		}
		return decimal.Zero, fmt.Errorf("calcular saldo da conta: %w", err)
	}
	return balance, nil
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

// DefaultAccountName é o nome da conta criada automaticamente para o usuário
// que ainda não possui nenhuma conta ativa.
const DefaultAccountName = "Carteira"

// EnsureDefault devolve o id da conta padrão do usuário: a conta ativa mais
// antiga. Se não houver nenhuma, cria a "Carteira" com saldo inicial zero.
// O advisory lock por usuário serializa chamadas simultâneas, impedindo que
// duas requisições concorrentes criem duas contas padrão.
func (r *Repository) EnsureDefault(ctx context.Context, userID string) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("iniciar transação: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('default_account:' || $1))`, userID); err != nil {
		return "", fmt.Errorf("bloquear conta padrão: %w", err)
	}

	var id string
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM financial_accounts
		WHERE user_id = $1 AND status = 'active'
		ORDER BY created_at ASC, id ASC
		LIMIT 1
	`, userID).Scan(&id)
	if err == nil {
		return id, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("buscar conta padrão: %w", err)
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO financial_accounts (user_id, name, type, currency, initial_balance, initial_balance_date, status)
		VALUES ($1, $2, 'carteira', 'BRL', 0, CURRENT_DATE, 'active')
		RETURNING id
	`, userID, DefaultAccountName).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("criar conta padrão: %w", err)
	}
	return id, tx.Commit()
}
