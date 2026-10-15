package investment

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestMarketValueAfterMovement(t *testing.T) {
	d := decimal.RequireFromString
	inv := &Investment{MarketValue: d("30000.00"), InvestedAmount: d("30000.00")}
	cases := []struct {
		name, typ, amount, want string
	}{
		{"aporte soma o dinheiro aplicado", TxContribution, "1000.00", "31000.00"},
		{"resgate retira o dinheiro sacado", TxRedemption, "2500.50", "27499.50"},
		{"resgate maior que a posição não fica negativo", TxRedemption, "40000.00", "0.00"},
		{"rendimento pago não altera a posição", TxIncome, "120.00", "30000.00"},
		{"taxa não altera a posição", TxFee, "15.00", "30000.00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MarketValueAfter(inv, MovementInput{Type: tc.typ, Amount: d(tc.amount)})
			if got.StringFixed(2) != tc.want {
				t.Fatalf("valor de mercado = %s, esperado %s", got.StringFixed(2), tc.want)
			}
		})
	}
}
