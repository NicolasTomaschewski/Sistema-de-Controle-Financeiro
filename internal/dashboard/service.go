// Package dashboard consolida indicadores de várias dimensões (fluxo
// mensal, saldo de contas, patrimônio líquido, carteira de investimentos)
// em uma visão única para a tela inicial da aplicação. Nunca calcula nada
// por conta própria — delega a cada domínio o cálculo de sua própria
// métrica, para que o dashboard nunca diverja dos dados detalhados.
package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/nicolastomaschewski/controle-financeiro/internal/account"
	"github.com/nicolastomaschewski/controle-financeiro/internal/investment"
	"github.com/nicolastomaschewski/controle-financeiro/internal/networth"
	"github.com/nicolastomaschewski/controle-financeiro/internal/transaction"
)

type Service struct {
	accountSvc   *account.Service
	transactions *transaction.Service
	netWorth     *networth.Service
	investments  *investment.Service
}

func NewService(accountSvc *account.Service, transactions *transaction.Service, netWorth *networth.Service, investments *investment.Service) *Service {
	return &Service{accountSvc: accountSvc, transactions: transactions, netWorth: netWorth, investments: investments}
}

type Summary struct {
	AccountsBalance    decimal.Decimal
	MonthlyIncome      decimal.Decimal
	MonthlyExpense     decimal.Decimal
	MonthlyResult      decimal.Decimal
	NetWorth           networth.Snapshot
	TotalInvested      decimal.Decimal
	TotalMarketValue   decimal.Decimal
	RecentTransactions []*transaction.Transaction
}

// Summary monta a visão consolidada do dashboard para o período informado.
func (s *Service) Summary(ctx context.Context, userID string, year int, month time.Month) (Summary, error) {
	accounts, err := s.accountSvc.List(ctx, userID, false)
	if err != nil {
		return Summary{}, fmt.Errorf("listar contas: %w", err)
	}
	accountsBalance := decimal.Zero
	for _, a := range accounts {
		accountsBalance = accountsBalance.Add(a.Balance)
	}

	report, err := s.transactions.MonthlyReport(ctx, userID, year, month, false)
	if err != nil {
		return Summary{}, fmt.Errorf("calcular resumo mensal: %w", err)
	}

	snap, err := s.netWorth.Calculate(ctx, userID)
	if err != nil {
		return Summary{}, fmt.Errorf("calcular patrimônio líquido: %w", err)
	}

	invSummary, err := s.investments.Summary(ctx, userID)
	if err != nil {
		return Summary{}, fmt.Errorf("calcular resumo de investimentos: %w", err)
	}

	recent, _, err := s.transactions.List(ctx, userID, transaction.Filter{Limit: 10})
	if err != nil {
		return Summary{}, fmt.Errorf("listar movimentações recentes: %w", err)
	}

	return Summary{
		AccountsBalance:    accountsBalance,
		MonthlyIncome:      report.Summary.TotalIncome,
		MonthlyExpense:     report.Summary.TotalExpense,
		MonthlyResult:      report.Summary.Result,
		NetWorth:           snap,
		TotalInvested:      invSummary.TotalInvested,
		TotalMarketValue:   invSummary.TotalMarketValue,
		RecentTransactions: recent,
	}, nil
}

type MonthlyFlowPoint struct {
	Year         int
	Month        int
	TotalIncome  decimal.Decimal
	TotalExpense decimal.Decimal
	Result       decimal.Decimal
}

// MonthlyFlow retorna a evolução mensal de receitas e despesas para os
// últimos `months` meses (incluindo o mês de referência), em ordem
// cronológica.
func (s *Service) MonthlyFlow(ctx context.Context, userID string, referenceYear int, referenceMonth time.Month, months int) ([]MonthlyFlowPoint, error) {
	if months <= 0 || months > 24 {
		months = 6
	}

	points := make([]MonthlyFlowPoint, months)
	cursor := time.Date(referenceYear, referenceMonth, 1, 0, 0, 0, 0, time.UTC)

	for i := months - 1; i >= 0; i-- {
		report, err := s.transactions.MonthlyReport(ctx, userID, cursor.Year(), cursor.Month(), false)
		if err != nil {
			return nil, fmt.Errorf("calcular fluxo de %04d-%02d: %w", cursor.Year(), cursor.Month(), err)
		}
		points[i] = MonthlyFlowPoint{
			Year: cursor.Year(), Month: int(cursor.Month()),
			TotalIncome: report.Summary.TotalIncome, TotalExpense: report.Summary.TotalExpense, Result: report.Summary.Result,
		}
		cursor = cursor.AddDate(0, -1, 0)
	}

	return points, nil
}

// ExpensesByCategory retorna a distribuição de despesas pagas por categoria
// no período informado.
func (s *Service) ExpensesByCategory(ctx context.Context, userID string, year int, month time.Month) ([]transaction.CategoryBreakdownItem, error) {
	report, err := s.transactions.MonthlyReport(ctx, userID, year, month, false)
	if err != nil {
		return nil, err
	}
	return report.ExpensesByCategory, nil
}
