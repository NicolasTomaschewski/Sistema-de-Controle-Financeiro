package recurring

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func ptr[T any](v T) *T { return &v }

func date(y int, m time.Month, day int) time.Time { return time.Date(y, m, day, 0, 0, 0, 0, time.UTC) }

func TestAnnualCostAndMonthlyEquivalent(t *testing.T) {
	cases := []struct {
		name        string
		periodicity string
		amount      string
		annual      string
		monthly     string
	}{
		{"mensal", PeriodicityMonthly, "1500.00", "18000.00", "1500.00"},
		{"anual", PeriodicityYearly, "1200.00", "1200.00", "100.00"},
		{"anual com dízima arredonda meio para cima", PeriodicityYearly, "1000.00", "1000.00", "83.33"},
		{"anual com centavos", PeriodicityYearly, "100.30", "100.30", "8.36"},
		{"semanal", PeriodicityWeekly, "50.00", "2600.00", "216.67"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Rule{Periodicity: tc.periodicity, PlannedAmount: d(tc.amount)}
			if got := r.AnnualCost().StringFixed(2); got != tc.annual {
				t.Errorf("AnnualCost = %s, esperado %s", got, tc.annual)
			}
			if got := r.MonthlyEquivalent().StringFixed(2); got != tc.monthly {
				t.Errorf("MonthlyEquivalent = %s, esperado %s", got, tc.monthly)
			}
		})
	}
}

func TestValidateDueMonth(t *testing.T) {
	if err := ValidateDueMonth(PeriodicityYearly, nil); err != ErrInvalidDueMonth {
		t.Error("regra anual sem mês de vencimento deveria ser rejeitada")
	}
	if err := ValidateDueMonth(PeriodicityYearly, ptr(13)); err != ErrInvalidDueMonth {
		t.Error("mês 13 deveria ser rejeitado")
	}
	if err := ValidateDueMonth(PeriodicityYearly, ptr(3)); err != nil {
		t.Errorf("mês 3 em regra anual deveria ser aceito: %v", err)
	}
	if err := ValidateDueMonth(PeriodicityMonthly, ptr(3)); err != ErrInvalidDueMonth {
		t.Error("regra mensal não deve aceitar mês de vencimento")
	}
	if err := ValidateDueMonth(PeriodicityMonthly, nil); err != nil {
		t.Errorf("regra mensal sem mês deveria ser aceita: %v", err)
	}
}

func TestIsDueInMonth(t *testing.T) {
	yearly := &Rule{Periodicity: PeriodicityYearly, DueMonth: ptr(3)}
	if !yearly.IsDueInMonth(time.March) || yearly.IsDueInMonth(time.April) {
		t.Error("regra anual deve vencer apenas no mês de vencimento")
	}
	monthly := &Rule{Periodicity: PeriodicityMonthly}
	if !monthly.IsDueInMonth(time.April) {
		t.Error("regra mensal deve vencer todo mês")
	}
}

func TestOccurrenceDateClampsToMonthEnd(t *testing.T) {
	r := &Rule{DueDay: ptr(31)}
	if got := r.OccurrenceDate(2026, time.February); !got.Equal(date(2026, time.February, 28)) {
		t.Errorf("dia 31 em fevereiro deveria virar 28, obteve %s", got)
	}
}

func TestSummarizeCosts(t *testing.T) {
	moradia, lazer := "cat-moradia", "cat-lazer"
	on := date(2026, time.October, 15)
	rules := []*Rule{
		{Periodicity: PeriodicityMonthly, PlannedAmount: d("1500.00"), CategoryID: &moradia, Active: true},
		{Periodicity: PeriodicityYearly, PlannedAmount: d("1000.00"), CategoryID: &moradia, DueMonth: ptr(2), Active: true},
		{Periodicity: PeriodicityMonthly, PlannedAmount: d("49.90"), CategoryID: &lazer, Active: true},
		// Não entram: inativa e encerrada antes da data de referência.
		{Periodicity: PeriodicityMonthly, PlannedAmount: d("999.00"), CategoryID: &lazer, Active: false},
		{Periodicity: PeriodicityMonthly, PlannedAmount: d("888.00"), CategoryID: &lazer, Active: true, EndDate: ptr(date(2026, time.September, 30))},
	}

	s := SummarizeCosts(rules, on)

	check := func(name string, got decimal.Decimal, want string) {
		t.Helper()
		if got.StringFixed(2) != want {
			t.Errorf("%s = %s, esperado %s", name, got.StringFixed(2), want)
		}
	}
	check("MonthlyRulesTotal", s.MonthlyRulesTotal, "1549.90")
	check("YearlyRulesTotal", s.YearlyRulesTotal, "1000.00")
	// 1500*12 + 1000 + 49.90*12 = 18000 + 1000 + 598.80
	check("AnnualTotal", s.AnnualTotal, "19598.80")
	// 19598.80 / 12 = 1633.2333... -> 1633.23 (do total exato, não da soma arredondada)
	check("MonthlyEquivalent", s.MonthlyEquivalent, "1633.23")

	if len(s.ByCategory) != 2 {
		t.Fatalf("esperava 2 categorias, obteve %d", len(s.ByCategory))
	}
	if *s.ByCategory[0].CategoryID != moradia {
		t.Error("categorias devem vir ordenadas pelo maior custo anual")
	}
	check("moradia anual", s.ByCategory[0].AnnualTotal, "19000.00")
	check("moradia mensal", s.ByCategory[0].MonthlyEquivalent, "1583.33")
	check("lazer anual", s.ByCategory[1].AnnualTotal, "598.80")
}

func TestSummarizeCostsEmpty(t *testing.T) {
	s := SummarizeCosts(nil, date(2026, time.January, 1))
	if !s.AnnualTotal.IsZero() || !s.MonthlyEquivalent.IsZero() || len(s.ByCategory) != 0 {
		t.Error("sem regras, o resumo deve ser zerado")
	}
}
