// Package asset implementa os bens patrimoniais do usuário (imóveis,
// veículos, participações, etc.) e seu histórico de avaliações.
package asset

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Asset struct {
	ID            string
	UserID        string
	Name          string
	Type          string
	CurrentValue  decimal.Decimal
	ValuationDate time.Time
	Notes         *string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Valuation struct {
	ID            string
	AssetID       string
	Value         decimal.Decimal
	ValuationDate time.Time
	Notes         *string
	CreatedAt     time.Time
}

const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

var (
	ErrNotFound        = errors.New("ativo não encontrado")
	ErrInvalidName     = errors.New("nome do ativo inválido")
	ErrInvalidType     = errors.New("tipo do ativo inválido")
	ErrInvalidValue    = errors.New("valor do ativo inválido")
	ErrValuationExists = errors.New("já existe uma avaliação para esta data")
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

func ValidateValue(v decimal.Decimal) error {
	if v.IsNegative() {
		return ErrInvalidValue
	}
	return nil
}
