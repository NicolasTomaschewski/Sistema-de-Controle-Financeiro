// Package investment implementa o cadastro de investimentos, suas
// movimentações (aportes, resgates, rendimentos, taxas) e métricas
// financeiras, sem depender de integração externa com corretoras.
package investment

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Investment struct {
	ID                string
	UserID            string
	Name              string
	AssetType         string
	Institution       *string
	Ticker            *string
	Quantity          *decimal.Decimal
	AveragePrice      *decimal.Decimal
	InvestedAmount    decimal.Decimal
	MarketValue       decimal.Decimal
	PositionDate      time.Time
	AcquisitionDate   *time.Time
	MaturityDate      *time.Time
	IncludeInNetWorth bool
	Notes             *string
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

const (
	TxContribution = "contribution"
	TxRedemption   = "redemption"
	TxIncome       = "income"
	TxFee          = "fee"

	StatusActive = "active"
	StatusClosed = "closed"
)

type Movement struct {
	ID           string
	UserID       string
	InvestmentID string
	AccountID    *string
	Type         string
	Amount       decimal.Decimal
	Quantity     *decimal.Decimal
	Fees         decimal.Decimal
	RealizedGain *decimal.Decimal
	OccurredOn   time.Time
	Notes        *string
	CreatedAt    time.Time
}

type Valuation struct {
	ID           string
	InvestmentID string
	MarketValue  decimal.Decimal
	PositionDate time.Time
	Notes        *string
	CreatedAt    time.Time
}

var (
	ErrNotFound             = errors.New("investimento não encontrado")
	ErrMovementNotFound     = errors.New("movimentação não encontrada")
	ErrInvalidName          = errors.New("nome do investimento inválido")
	ErrInvalidAssetType     = errors.New("tipo de ativo inválido")
	ErrInvalidAmount        = errors.New("valor inválido")
	ErrInvalidQuantity      = errors.New("quantidade inválida")
	ErrInvalidMovementType  = errors.New("tipo de movimentação inválido")
	ErrAccountNotOwned      = errors.New("conta não pertence ao usuário autenticado")
	ErrInsufficientQuantity = errors.New("quantidade insuficiente na posição para este resgate")
	ErrValuationExists      = errors.New("já existe um valor de mercado registrado para esta data")
)

func ValidateName(name string) error {
	if len(strings.TrimSpace(name)) == 0 || len(name) > 150 {
		return ErrInvalidName
	}
	return nil
}

func ValidateAssetType(t string) error {
	if len(strings.TrimSpace(t)) == 0 || len(t) > 50 {
		return ErrInvalidAssetType
	}
	return nil
}

func ValidateAmount(a decimal.Decimal) error {
	if a.IsZero() || a.IsNegative() {
		return ErrInvalidAmount
	}
	return nil
}

func ValidateMovementType(t string) error {
	switch t {
	case TxContribution, TxRedemption, TxIncome, TxFee:
		return nil
	default:
		return ErrInvalidMovementType
	}
}

// UnrealizedGain é a diferença entre o valor de mercado atual e o valor
// investido (custo). Representa ganho ou perda não realizado — ainda não
// efetivado por venda/resgate.
func (i *Investment) UnrealizedGain() decimal.Decimal {
	return i.MarketValue.Sub(i.InvestedAmount)
}

// SimpleReturnPercent calcula a rentabilidade simples (%) com base no valor
// investido atual. Retorna nil quando não há valor investido (divisão por
// zero), para não apresentar uma rentabilidade fictícia.
func (i *Investment) SimpleReturnPercent() *decimal.Decimal {
	if i.InvestedAmount.IsZero() {
		return nil
	}
	pct := i.UnrealizedGain().Div(i.InvestedAmount).Mul(decimal.NewFromInt(100))
	return &pct
}
