// Package liability implementa as obrigações financeiras do usuário
// (financiamentos, empréstimos, dívidas pessoais) e o histórico de baixas
// de saldo devedor.
package liability

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Liability struct {
	ID              string
	UserID          string
	Name            string
	Type            string
	PrincipalAmount decimal.Decimal
	CurrentBalance  decimal.Decimal
	ReferenceDate   time.Time
	Notes           *string
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Valuation struct {
	ID            string
	LiabilityID   string
	Balance       decimal.Decimal
	ReferenceDate time.Time
	Notes         *string
	CreatedAt     time.Time
}

const (
	StatusActive  = "active"
	StatusSettled = "settled"
)

var (
	ErrNotFound        = errors.New("passivo não encontrado")
	ErrInvalidName     = errors.New("nome do passivo inválido")
	ErrInvalidType     = errors.New("tipo do passivo inválido")
	ErrInvalidAmount   = errors.New("valor do passivo inválido")
	ErrValuationExists = errors.New("já existe um registro de saldo para esta data")
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

func ValidateAmount(v decimal.Decimal) error {
	if v.IsNegative() {
		return ErrInvalidAmount
	}
	return nil
}
