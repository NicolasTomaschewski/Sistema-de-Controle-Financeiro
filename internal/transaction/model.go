// Package transaction implementa os lançamentos financeiros (receitas e
// despesas) e as transferências entre contas do mesmo usuário.
package transaction

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	TypeIncome      = "income"
	TypeExpense     = "expense"
	TypeTransferIn  = "transfer_in"
	TypeTransferOut = "transfer_out"

	StatusPlanned   = "planned"
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusCancelled = "cancelled"
)

type Transaction struct {
	ID                      string
	UserID                  string
	AccountID               string
	CategoryID              *string
	TransferID              *string
	RecurringRuleID         *string
	InvestmentTransactionID *string
	Type                    string
	Status                  string
	Amount                  decimal.Decimal
	Description             string
	OccurredOn              time.Time
	Notes                   *string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

var (
	ErrNotFound             = errors.New("lançamento não encontrado")
	ErrInvalidAmount        = errors.New("valor inválido")
	ErrInvalidDescription   = errors.New("descrição inválida")
	ErrInvalidType          = errors.New("tipo de lançamento inválido")
	ErrInvalidStatus        = errors.New("status de lançamento inválido")
	ErrAccountNotOwned      = errors.New("conta não pertence ao usuário autenticado")
	ErrCategoryNotVisible   = errors.New("categoria não encontrada ou não pertence ao usuário")
	ErrCategoryKindMismatch = errors.New("categoria incompatível com o tipo do lançamento")
	ErrIsTransferLeg        = errors.New("movimentações de transferência não podem ser editadas individualmente")
)

func ValidateAmount(a decimal.Decimal) error {
	if a.IsZero() || a.IsNegative() {
		return ErrInvalidAmount
	}
	return nil
}

func ValidateDescription(d string) error {
	if len(strings.TrimSpace(d)) == 0 || len(d) > 255 {
		return ErrInvalidDescription
	}
	return nil
}

func ValidateType(t string) error {
	if t != TypeIncome && t != TypeExpense {
		return ErrInvalidType
	}
	return nil
}

func ValidateStatus(s string) error {
	switch s {
	case StatusPlanned, StatusPending, StatusPaid, StatusCancelled:
		return nil
	default:
		return ErrInvalidStatus
	}
}

// IsTransferLeg indica se o lançamento foi gerado por uma transferência
// entre contas, caso em que não deve ser editado ou excluído isoladamente.
func (t *Transaction) IsTransferLeg() bool {
	return t.TransferID != nil
}
