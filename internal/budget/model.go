// Package budget implementa o orçamento mensal por categoria: valor
// planejado, gasto realizado, saldo disponível e percentual consumido.
package budget

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type Budget struct {
	ID            string
	UserID        string
	CategoryID    string
	Year          int
	Month         int
	PlannedAmount decimal.Decimal
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

var (
	ErrNotFound      = errors.New("orçamento não encontrado")
	ErrInvalidAmount = errors.New("valor planejado inválido")
	ErrInvalidPeriod = errors.New("período inválido")
	ErrAlreadyExists = errors.New("já existe um orçamento para esta categoria e período")
)

func ValidateAmount(a decimal.Decimal) error {
	if a.IsZero() || a.IsNegative() {
		return ErrInvalidAmount
	}
	return nil
}

func ValidatePeriod(year, month int) error {
	if year < 2000 || year > 2100 || month < 1 || month > 12 {
		return ErrInvalidPeriod
	}
	return nil
}

type Summary struct {
	CategoryID      string
	CategoryName    string
	PlannedAmount   decimal.Decimal
	SpentAmount     decimal.Decimal
	AvailableAmount decimal.Decimal
	PercentUsed     decimal.Decimal
	OverLimit       bool
}
