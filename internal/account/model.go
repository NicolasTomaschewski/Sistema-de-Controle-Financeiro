// Package account implementa as contas financeiras do usuário (carteira,
// conta corrente, poupança, etc.) e o cálculo de saldo a partir das
// movimentações efetivadas.
package account

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Account struct {
	ID                 string
	UserID             string
	Name               string
	Type               string
	Institution        *string
	Currency           string
	InitialBalance     decimal.Decimal
	InitialBalanceDate time.Time
	Status             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

var (
	ErrNotFound        = errors.New("conta não encontrada")
	ErrInvalidName     = errors.New("nome da conta inválido")
	ErrInvalidType     = errors.New("tipo da conta inválido")
	ErrInvalidCurrency = errors.New("moeda inválida")
	ErrHasTransactions = errors.New("conta possui movimentações vinculadas")
)

func ValidateName(name string) error {
	if len(strings.TrimSpace(name)) == 0 || len(name) > 150 {
		return ErrInvalidName
	}
	return nil
}

func ValidateType(t string) error {
	if len(strings.TrimSpace(t)) == 0 || len(t) > 50 {
		return ErrInvalidType
	}
	return nil
}

func ValidateCurrency(c string) error {
	if len(c) != 3 {
		return ErrInvalidCurrency
	}
	return nil
}
