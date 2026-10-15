// Package recurring implementa regras de despesas recorrentes e a geração
// (materialização) das ocorrências previstas como lançamentos identificáveis.
package recurring

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	PeriodicityMonthly = "monthly"
	PeriodicityWeekly  = "weekly"
	PeriodicityYearly  = "yearly"
)

type Rule struct {
	ID            string
	UserID        string
	AccountID     *string
	CategoryID    *string
	Description   string
	PlannedAmount decimal.Decimal
	Periodicity   string
	StartDate     time.Time
	EndDate       *time.Time
	DueDay        *int
	DueMonth      *int // obrigatório apenas para regras anuais
	Active        bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

var (
	ErrNotFound        = errors.New("regra de recorrência não encontrada")
	ErrInvalidDesc     = errors.New("descrição inválida")
	ErrInvalidAmount   = errors.New("valor planejado inválido")
	ErrInvalidPeriod   = errors.New("periodicidade inválida")
	ErrInvalidDueDay   = errors.New("dia de vencimento inválido")
	ErrInvalidDueMonth = errors.New("mês de vencimento inválido")
	ErrNotDueInMonth   = errors.New("a regra anual não vence no mês informado")
	ErrRuleInactive    = errors.New("regra de recorrência está inativa")
	ErrOutsideValidity = errors.New("período fora da vigência da regra")
)

func ValidateDescription(d string) error {
	if len(strings.TrimSpace(d)) == 0 || len(d) > 255 {
		return ErrInvalidDesc
	}
	return nil
}

func ValidateAmount(a decimal.Decimal) error {
	if a.IsZero() || a.IsNegative() {
		return ErrInvalidAmount
	}
	return nil
}

func ValidatePeriodicity(p string) error {
	switch p {
	case PeriodicityMonthly, PeriodicityWeekly, PeriodicityYearly:
		return nil
	default:
		return ErrInvalidPeriod
	}
}

func ValidateDueDay(d *int) error {
	if d == nil {
		return nil
	}
	if *d < 1 || *d > 31 {
		return ErrInvalidDueDay
	}
	return nil
}

// ValidateDueMonth exige o mês de vencimento nas regras anuais e o proíbe nas
// demais, para que a periodicidade determine sem ambiguidade quando a regra vence.
func ValidateDueMonth(periodicity string, m *int) error {
	if periodicity != PeriodicityYearly {
		if m != nil {
			return ErrInvalidDueMonth
		}
		return nil
	}
	if m == nil || *m < 1 || *m > 12 {
		return ErrInvalidDueMonth
	}
	return nil
}

// OccurrenceDate calcula a data de vencimento da ocorrência de uma regra
// para o ano/mês informados, ajustando para o último dia do mês
// quando o dia de vencimento não existir nesse mês (ex.: dia 31 em
// fevereiro).
func (r *Rule) OccurrenceDate(year int, month time.Month) time.Time {
	day := 1
	if r.DueDay != nil {
		day = *r.DueDay
	}
	firstOfNextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	lastDayOfMonth := firstOfNextMonth.AddDate(0, 0, -1).Day()
	if day > lastDayOfMonth {
		day = lastDayOfMonth
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// IsWithinValidity indica se a data informada está dentro da vigência da
// regra (entre start_date e end_date, quando definida).
func (r *Rule) IsWithinValidity(date time.Time) bool {
	if date.Before(r.StartDate) {
		return false
	}
	if r.EndDate != nil && date.After(*r.EndDate) {
		return false
	}
	return true
}

// IsDueInMonth indica se a regra gera ocorrência no mês informado: regras
// mensais vencem todo mês; regras anuais, apenas no mês de vencimento.
func (r *Rule) IsDueInMonth(month time.Month) bool {
	if r.Periodicity == PeriodicityYearly {
		return r.DueMonth != nil && time.Month(*r.DueMonth) == month
	}
	return true
}

// AnnualCost é o custo da regra em um ano completo: 12 ocorrências para
// regras mensais, 52 para semanais e 1 para anuais. Valor exato, sem arredondamento.
func (r *Rule) AnnualCost() decimal.Decimal {
	switch r.Periodicity {
	case PeriodicityMonthly:
		return r.PlannedAmount.Mul(decimal.NewFromInt(12))
	case PeriodicityWeekly:
		return r.PlannedAmount.Mul(decimal.NewFromInt(52))
	default:
		return r.PlannedAmount
	}
}

// MonthlyEquivalent distribui o custo anual em 12 meses, arredondado a
// 2 casas (meio para cima). Serve para comparar custos de periodicidades
// diferentes; nunca é usado para gerar lançamentos.
func (r *Rule) MonthlyEquivalent() decimal.Decimal {
	return r.AnnualCost().Div(decimal.NewFromInt(12)).Round(2)
}

// IsInEffect indica se a regra está ativa e vigente na data informada
// (considera apenas o término; regras com início futuro já contam como custo previsto).
func (r *Rule) IsInEffect(on time.Time) bool {
	return r.Active && (r.EndDate == nil || !r.EndDate.Before(on))
}

// CostSummary consolida o custo de vida fixo a partir das regras vigentes.
type CostSummary struct {
	MonthlyRulesTotal decimal.Decimal // soma dos valores das regras mensais
	YearlyRulesTotal  decimal.Decimal // soma dos valores das regras anuais
	AnnualTotal       decimal.Decimal // custo fixo total de um ano
	MonthlyEquivalent decimal.Decimal // AnnualTotal / 12, arredondado a 2 casas
	ByCategory        []CategoryCost
}

type CategoryCost struct {
	CategoryID        *string
	AnnualTotal       decimal.Decimal
	MonthlyEquivalent decimal.Decimal
}

// SummarizeCosts calcula o resumo apenas com regras vigentes na data
// informada. O equivalente mensal é derivado do total anual exato (e não da
// soma de valores já arredondados), evitando acúmulo de erro de centavos.
func SummarizeCosts(rules []*Rule, on time.Time) CostSummary {
	twelve := decimal.NewFromInt(12)
	sum := CostSummary{MonthlyRulesTotal: decimal.Zero, YearlyRulesTotal: decimal.Zero, AnnualTotal: decimal.Zero}
	byCat := map[string]*CategoryCost{}
	var order []string

	for _, r := range rules {
		if !r.IsInEffect(on) {
			continue
		}
		switch r.Periodicity {
		case PeriodicityMonthly:
			sum.MonthlyRulesTotal = sum.MonthlyRulesTotal.Add(r.PlannedAmount)
		case PeriodicityYearly:
			sum.YearlyRulesTotal = sum.YearlyRulesTotal.Add(r.PlannedAmount)
		}
		annual := r.AnnualCost()
		sum.AnnualTotal = sum.AnnualTotal.Add(annual)

		key := ""
		if r.CategoryID != nil {
			key = *r.CategoryID
		}
		c, ok := byCat[key]
		if !ok {
			c = &CategoryCost{CategoryID: r.CategoryID, AnnualTotal: decimal.Zero}
			byCat[key] = c
			order = append(order, key)
		}
		c.AnnualTotal = c.AnnualTotal.Add(annual)
	}

	sum.MonthlyEquivalent = sum.AnnualTotal.Div(twelve).Round(2)
	for _, key := range order {
		c := byCat[key]
		c.MonthlyEquivalent = c.AnnualTotal.Div(twelve).Round(2)
		sum.ByCategory = append(sum.ByCategory, *c)
	}
	sort.SliceStable(sum.ByCategory, func(i, j int) bool {
		return sum.ByCategory[i].AnnualTotal.GreaterThan(sum.ByCategory[j].AnnualTotal)
	})
	return sum
}
