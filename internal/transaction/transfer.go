package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/nicolastomaschewski/controle-financeiro/internal/account"
)

var (
	ErrTransferSameAccount = errors.New("a conta de origem e de destino devem ser diferentes")
	ErrTransferNotFound    = errors.New("transferência não encontrada")
)

type Transfer struct {
	ID            string
	UserID        string
	FromAccountID string
	ToAccountID   string
	Amount        decimal.Decimal
	OccurredOn    time.Time
	Description   string
	CreatedAt     time.Time
}

type TransferInput struct {
	FromAccountID string
	ToAccountID   string
	Amount        decimal.Decimal
	OccurredOn    time.Time
	Description   string
}

// CreateTransfer registra uma transferência entre duas contas do mesmo
// usuário de forma atômica: a transferência e as duas movimentações
// vinculadas (débito na origem, crédito no destino) são gravadas na mesma
// transação SQL — se qualquer etapa falhar, nada é persistido.
func (s *Service) CreateTransfer(ctx context.Context, userID string, in TransferInput) (*Transfer, error) {
	if in.FromAccountID == in.ToAccountID {
		return nil, ErrTransferSameAccount
	}
	if err := ValidateAmount(in.Amount); err != nil {
		return nil, err
	}

	// Validação prévia de propriedade das contas, para retornar um erro de
	// negócio claro em vez de depender apenas da restrição composta no banco.
	if _, err := s.accounts.GetByID(ctx, in.FromAccountID, userID); err != nil {
		if errors.Is(err, account.ErrNotFound) {
			return nil, fmt.Errorf("conta de origem: %w", ErrAccountNotOwned)
		}
		return nil, err
	}
	if _, err := s.accounts.GetByID(ctx, in.ToAccountID, userID); err != nil {
		if errors.Is(err, account.ErrNotFound) {
			return nil, fmt.Errorf("conta de destino: %w", ErrAccountNotOwned)
		}
		return nil, err
	}

	description := in.Description
	if description == "" {
		description = "Transferência entre contas"
	}

	transfer, err := s.repo.createTransferTx(ctx, userID, in.FromAccountID, in.ToAccountID, in.Amount, in.OccurredOn, description)
	if err != nil {
		return nil, err
	}

	s.audit.Record(ctx, &userID, "transfer.created", "transfer", transfer.ID, map[string]any{
		"from_account_id": in.FromAccountID,
		"to_account_id":   in.ToAccountID,
		"amount":          in.Amount.String(),
	})
	return transfer, nil
}

func (r *Repository) createTransferTx(ctx context.Context, userID, fromAccountID, toAccountID string, amount decimal.Decimal, occurredOn time.Time, description string) (*Transfer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback()

	t := &Transfer{UserID: userID, FromAccountID: fromAccountID, ToAccountID: toAccountID, Amount: amount, OccurredOn: occurredOn, Description: description}
	const insertTransfer = `
		INSERT INTO transfers (user_id, from_account_id, to_account_id, amount, occurred_on, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	if err := tx.QueryRowContext(ctx, insertTransfer, userID, fromAccountID, toAccountID, amount, occurredOn, description).Scan(&t.ID, &t.CreatedAt); err != nil {
		return nil, fmt.Errorf("registrar transferência: %w", err)
	}

	const insertLeg = `
		INSERT INTO transactions (user_id, account_id, transfer_id, type, status, amount, description, occurred_on)
		VALUES ($1, $2, $3, $4, 'paid', $5, $6, $7)
	`
	if _, err := tx.ExecContext(ctx, insertLeg, userID, fromAccountID, t.ID, TypeTransferOut, amount, "Transferência enviada: "+description, occurredOn); err != nil {
		return nil, fmt.Errorf("registrar movimentação de saída: %w", err)
	}
	if _, err := tx.ExecContext(ctx, insertLeg, userID, toAccountID, t.ID, TypeTransferIn, amount, "Transferência recebida: "+description, occurredOn); err != nil {
		return nil, fmt.Errorf("registrar movimentação de entrada: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmar transferência: %w", err)
	}
	return t, nil
}

func (s *Service) ListTransfers(ctx context.Context, userID string, limit, offset int) ([]*Transfer, error) {
	return s.repo.listTransfers(ctx, userID, limit, offset)
}

func (r *Repository) listTransfers(ctx context.Context, userID string, limit, offset int) ([]*Transfer, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	const q = `
		SELECT id, user_id, from_account_id, to_account_id, amount, occurred_on, description, created_at
		FROM transfers WHERE user_id = $1
		ORDER BY occurred_on DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listar transferências: %w", err)
	}
	defer rows.Close()

	var out []*Transfer
	for rows.Next() {
		var t Transfer
		if err := rows.Scan(&t.ID, &t.UserID, &t.FromAccountID, &t.ToAccountID, &t.Amount, &t.OccurredOn, &t.Description, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("ler transferência: %w", err)
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}
