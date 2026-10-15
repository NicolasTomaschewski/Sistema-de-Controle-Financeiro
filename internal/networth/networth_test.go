package networth

import (
	"testing"

	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestBuildCompositionGroupsByTypeAndComputesShares(t *testing.T) {
	c := BuildComposition([]Slice{
		{Group: GroupAsset, Type: "imovel", Value: d("300000.00")},
		{Group: GroupAsset, Type: "imovel", Value: d("150000.00")},
		{Group: GroupAsset, Type: "veiculo", Value: d("50000.00")},
		{Group: GroupInvestment, Type: "acoes", Value: d("33333.33")},
		// Mesmo nome de tipo em grupos diferentes não pode ser mesclado.
		{Group: GroupInvestment, Type: "veiculo", Value: d("100.00")},
	})

	if got := c.AssetsValue.StringFixed(2); got != "500000.00" {
		t.Errorf("AssetsValue = %s", got)
	}
	if got := c.InvestmentsValue.StringFixed(2); got != "33433.33" {
		t.Errorf("InvestmentsValue = %s", got)
	}
	if got := c.Total.StringFixed(2); got != "533433.33" {
		t.Errorf("Total = %s", got)
	}
	if len(c.Slices) != 4 {
		t.Fatalf("esperava 4 fatias, obteve %d", len(c.Slices))
	}
	first := c.Slices[0]
	if first.Type != "imovel" || first.Value.StringFixed(2) != "450000.00" {
		t.Errorf("a maior fatia deveria ser imóveis com 450000.00, obteve %s %s", first.Type, first.Value)
	}
	// 450000 / 533433.33 = 84.359...% -> 84.4
	if got := first.SharePercent.StringFixed(1); got != "84.4" {
		t.Errorf("participação de imóveis = %s, esperado 84.4", got)
	}
}

func TestBuildCompositionEmpty(t *testing.T) {
	c := BuildComposition(nil)
	if !c.Total.IsZero() || len(c.Slices) != 0 {
		t.Error("sem itens, a composição deve ser vazia e zerada")
	}
}
