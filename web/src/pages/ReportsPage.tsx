import { useEffect, useState } from 'react'
import { budgetsApi, dashboardApi, investmentsApi, netWorthApi } from '../api/endpoints'
import { ApiError } from '../api/client'
import type { BudgetSummary, InvestmentSummary, MonthlyFlowPoint, NetWorth } from '../api/types'
import { formatBRL, monthLabel } from '../utils/format'
import { MonthPicker } from '../components/MonthPicker'
import { ErrorState, LoadingState } from '../components/States'
import { GroupedBarChart } from '../components/BarChart'

export default function ReportsPage() {
  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)

  const [flow, setFlow] = useState<MonthlyFlowPoint[] | null>(null)
  const [expensesByCategory, setExpensesByCategory] = useState<{ category_name: string; total: string }[] | null>(null)
  const [budgets, setBudgets] = useState<BudgetSummary[] | null>(null)
  const [netWorth, setNetWorth] = useState<NetWorth | null>(null)
  const [investmentSummary, setInvestmentSummary] = useState<InvestmentSummary | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let active = true
    setError(null)
    Promise.all([
      dashboardApi.monthlyFlow(year, month, 12),
      dashboardApi.expensesByCategory(year, month),
      budgetsApi.summary(year, month),
      netWorthApi.get(),
      investmentsApi.summary(),
    ])
      .then(([f, e, b, nw, is]) => {
        if (!active) return
        setFlow(f); setExpensesByCategory(e); setBudgets(b); setNetWorth(nw); setInvestmentSummary(is)
      })
      .catch((err) => { if (active) setError(err instanceof ApiError ? err.message : 'Erro ao carregar relatórios.') })
    return () => { active = false }
  }, [year, month])

  const totalExpenses = expensesByCategory?.reduce((acc, c) => acc + Number(c.total), 0) ?? 0

  return (
    <div>
      <div className="page-header">
        <h1>Relatórios</h1>
      </div>

      <MonthPicker year={year} month={month} onChange={(y, m) => { setYear(y); setMonth(m) }} />
      {error && <ErrorState message={error} />}

      <div className="card">
        <h3>Receitas e despesas — últimos 12 meses</h3>
        {flow === null ? <LoadingState /> : (
          <GroupedBarChart
            groups={flow.map((p) => ({
              label: monthLabel(p.month).slice(0, 3),
              series: [
                { label: 'Receita', value: Number(p.total_income), color: 'var(--series-increase)' },
                { label: 'Despesa', value: Number(p.total_expense), color: 'var(--series-decrease)' },
              ],
            }))}
          />
        )}
      </div>

      <div className="grid grid--two">
        <div className="card">
          <h3>Despesas por categoria</h3>
          {expensesByCategory === null ? <LoadingState /> : expensesByCategory.length === 0 ? <p>Sem despesas no período.</p> : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.6rem' }}>
              {expensesByCategory.map((c) => {
                const pct = totalExpenses > 0 ? (Number(c.total) / totalExpenses) * 100 : 0
                return (
                  <div key={c.category_name}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem' }}>
                      <span>{c.category_name}</span>
                      <span>{formatBRL(c.total)}</span>
                    </div>
                    <div className="progress-bar"><div className="progress-bar__fill" style={{ width: `${pct}%` }} /></div>
                  </div>
                )
              })}
            </div>
          )}
        </div>

        <div className="card">
          <h3>Orçamento vs. realizado</h3>
          {budgets === null ? <LoadingState /> : budgets.length === 0 ? <p>Nenhum orçamento definido para este mês.</p> : (
            <table>
              <thead><tr><th>Categoria</th><th>Planejado</th><th>Gasto</th><th>Disponível</th></tr></thead>
              <tbody>
                {budgets.map((b) => (
                  <tr key={b.category_id}>
                    <td>{b.category_name}</td>
                    <td>{formatBRL(b.planned_amount)}</td>
                    <td>{formatBRL(b.spent_amount)}</td>
                    <td style={{ color: b.over_limit ? 'var(--color-danger)' : undefined }}>{formatBRL(b.available_amount)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      <div className="card">
        <h3>Composição da carteira de investimentos</h3>
        {investmentSummary === null ? <LoadingState /> : Object.keys(investmentSummary.by_asset_type).length === 0 ? <p>Nenhum investimento cadastrado.</p> : (
          <table>
            <thead><tr><th>Tipo</th><th>Valor de mercado</th></tr></thead>
            <tbody>
              {Object.entries(investmentSummary.by_asset_type).map(([type, value]) => (
                <tr key={type}><td>{type}</td><td>{formatBRL(value)}</td></tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="card">
        <h3>Composição patrimonial atual</h3>
        {netWorth === null ? <LoadingState /> : (
          <table>
            <thead><tr><th>Componente</th><th>Valor</th></tr></thead>
            <tbody>
              <tr><td>Saldo disponível</td><td>{formatBRL(netWorth.accounts_balance)}</td></tr>
              <tr><td>Ativos declarados</td><td>{formatBRL(netWorth.declared_assets_value)}</td></tr>
              <tr><td>Investimentos</td><td>{formatBRL(netWorth.investments_value)}</td></tr>
              <tr><td>Passivos</td><td>-{formatBRL(netWorth.total_liabilities)}</td></tr>
              <tr><td><strong>Patrimônio líquido</strong></td><td><strong>{formatBRL(netWorth.net_worth)}</strong></td></tr>
            </tbody>
          </table>
        )}
        <p style={{ fontSize: '0.78rem', color: 'var(--color-text-muted)', marginTop: '0.5rem' }}>
          Esta é uma fotografia da composição atual. O histórico detalhado de evolução patrimonial ao longo do
          tempo depende dos registros de avaliação de cada ativo/passivo/investimento individualmente.
        </p>
      </div>
    </div>
  )
}
