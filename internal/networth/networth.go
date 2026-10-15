// Package networth consolida contas financeiras, ativos declarados e
// passivos em uma única visão de patrimônio líquido, evitando dupla
// contabilização entre os módulos.
package networth

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"github.com/nicolastomaschewski/controle-financeiro/internal/account"
	"github.com/nicolastomaschewski/controle-financeiro/internal/asset"
	"github.com/nicolastomaschewski/controle-financeiro/internal/investment"
	"github.com/nicolastomaschewski/controle-financeiro/internal/liability"
)

type Service struct {
	accounts    *account.Repository
	assets      *asset.Repository
	liabilities *liability.Repository
	investments *investment.Repository
}

func NewService(accounts *account.Repository, assets *asset.Repository, liabilities *liability.Repository, investments *investment.Repository) *Service {
	return &Service{accounts: accounts, assets: assets, liabilities: liabilities, investments: investments}
}

// Snapshot representa a composição do patrimônio no momento do cálculo.
//
// Política de contabilização (documentada para evitar dupla contagem):
//   - AccountsBalance: soma dos saldos atuais das contas financeiras ativas
//     (dinheiro líquido/disponível).
//   - DeclaredAssetsValue: soma do valor atual dos ativos patrimoniais
//     ativos (imóveis, veículos, etc.), informados manualmente pelo usuário.
//   - InvestmentsValue: soma do valor de mercado dos investimentos ativos
//     cujo campo include_in_net_worth é verdadeiro. Esse campo permite ao
//     usuário excluir uma posição já representada em uma conta financeira
//     do tipo "investimento", evitando contar o mesmo dinheiro duas vezes —
//     a fonte de verdade de cada posição é sempre explícita.
//   - GrossWorth = AccountsBalance + DeclaredAssetsValue + InvestmentsValue.
//   - TotalLiabilities: soma do saldo devedor atual dos passivos ativos.
//   - NetWorth = GrossWorth - TotalLiabilities.
type Snapshot struct {
	AccountsBalance     decimal.Decimal
	DeclaredAssetsValue decimal.Decimal
	InvestmentsValue    decimal.Decimal
	GrossWorth          decimal.Decimal
	TotalLiabilities    decimal.Decimal
	NetWorth            decimal.Decimal
	AsOf                time.Time
}

func (s *Service) Calculate(ctx context.Context, userID string) (Snapshot, error) {
	accounts, err := s.accounts.List(ctx, userID, false)
	if err != nil {
		return Snapshot{}, fmt.Errorf("listar contas: %w", err)
	}
	accountsBalance := decimal.Zero
	for _, a := range accounts {
		balance, err := s.accounts.CurrentBalance(ctx, a.ID, userID)
		if err != nil {
			return Snapshot{}, fmt.Errorf("calcular saldo da conta %s: %w", a.ID, err)
		}
		accountsBalance = accountsBalance.Add(balance)
	}

	assets, err := s.assets.List(ctx, userID, false)
	if err != nil {
		return Snapshot{}, fmt.Errorf("listar ativos: %w", err)
	}
	assetsValue := decimal.Zero
	for _, a := range assets {
		assetsValue = assetsValue.Add(a.CurrentValue)
	}

	liabilities, err := s.liabilities.List(ctx, userID, false)
	if err != nil {
		return Snapshot{}, fmt.Errorf("listar passivos: %w", err)
	}
	totalLiabilities := decimal.Zero
	for _, l := range liabilities {
		totalLiabilities = totalLiabilities.Add(l.CurrentBalance)
	}

	investments, err := s.investments.List(ctx, userID, false)
	if err != nil {
		return Snapshot{}, fmt.Errorf("listar investimentos: %w", err)
	}
	investmentsValue := decimal.Zero
	for _, inv := range investments {
		if inv.IncludeInNetWorth {
			investmentsValue = investmentsValue.Add(inv.MarketValue)
		}
	}

	gross := accountsBalance.Add(assetsValue).Add(investmentsValue)

	return Snapshot{
		AccountsBalance:     accountsBalance,
		DeclaredAssetsValue: assetsValue,
		InvestmentsValue:    investmentsValue,
		GrossWorth:          gross,
		TotalLiabilities:    totalLiabilities,
		NetWorth:            gross.Sub(totalLiabilities),
		AsOf:                time.Now().UTC(),
	}, nil
}

// Slice é uma fatia da composição patrimonial: o valor somado de todos os
// bens (Group "asset") ou investimentos (Group "investment") de um mesmo tipo.
type Slice struct {
	Group        string
	Type         string
	Value        decimal.Decimal
	SharePercent decimal.Decimal // participação no total, arredondada a 1 casa
}

// Composition é a visão de patrimônio da tela Patrimônio: bens declarados e
// investimentos, sem contas financeiras nem passivos. Segue as mesmas regras
// de inclusão do Snapshot (ativos não arquivados; investimentos ativos com
// include_in_net_worth), então seus totais coincidem com DeclaredAssetsValue
// e InvestmentsValue.
type Composition struct {
	AssetsValue      decimal.Decimal
	InvestmentsValue decimal.Decimal
	Total            decimal.Decimal
	Slices           []Slice
	AsOf             time.Time
}

const (
	GroupAsset      = "asset"
	GroupInvestment = "investment"
)

func (s *Service) Composition(ctx context.Context, userID string) (Composition, error) {
	assets, err := s.assets.List(ctx, userID, false)
	if err != nil {
		return Composition{}, fmt.Errorf("listar ativos: %w", err)
	}
	investments, err := s.investments.List(ctx, userID, false)
	if err != nil {
		return Composition{}, fmt.Errorf("listar investimentos: %w", err)
	}

	var items []Slice
	for _, a := range assets {
		items = append(items, Slice{Group: GroupAsset, Type: a.Type, Value: a.CurrentValue})
	}
	for _, inv := range investments {
		if inv.IncludeInNetWorth {
			items = append(items, Slice{Group: GroupInvestment, Type: inv.AssetType, Value: inv.MarketValue})
		}
	}
	c := BuildComposition(items)
	c.AsOf = time.Now().UTC()
	return c, nil
}

// BuildComposition agrupa os itens por (grupo, tipo), calcula os totais e a
// participação de cada fatia, ordenando da maior para a menor.
func BuildComposition(items []Slice) Composition {
	c := Composition{AssetsValue: decimal.Zero, InvestmentsValue: decimal.Zero, Total: decimal.Zero}
	index := map[[2]string]int{}
	for _, it := range items {
		if it.Group == GroupAsset {
			c.AssetsValue = c.AssetsValue.Add(it.Value)
		} else {
			c.InvestmentsValue = c.InvestmentsValue.Add(it.Value)
		}
		key := [2]string{it.Group, it.Type}
		if i, ok := index[key]; ok {
			c.Slices[i].Value = c.Slices[i].Value.Add(it.Value)
			continue
		}
		index[key] = len(c.Slices)
		c.Slices = append(c.Slices, Slice{Group: it.Group, Type: it.Type, Value: it.Value})
	}
	c.Total = c.AssetsValue.Add(c.InvestmentsValue)

	for i := range c.Slices {
		c.Slices[i].SharePercent = decimal.Zero
		if c.Total.IsPositive() {
			c.Slices[i].SharePercent = c.Slices[i].Value.Div(c.Total).Mul(decimal.NewFromInt(100)).Round(1)
		}
	}
	sort.SliceStable(c.Slices, func(i, j int) bool { return c.Slices[i].Value.GreaterThan(c.Slices[j].Value) })
	return c
}
