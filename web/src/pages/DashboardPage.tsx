import { useEffect, useState } from 'react'
import { dashboardApi } from '../api/endpoints'
import { ApiError } from '../api/client'
import type { DashboardSummary, MonthlyFlowPoint } from '../api/types'
import { formatBRL, formatDateBR, monthLabel, TRANSACTION_STATUS_LABEL, TRANSACTION_TYPE_LABEL } from '../utils/format'
import { MonthPicker } from '../components/MonthPicker'
import { ErrorState, LoadingState } from '../components/States'
import { GroupedBarChart } from '../components/BarChart'

export default function DashboardPage() {
  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)

  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [flow, setFlow] = useState<MonthlyFlowPoint[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let active = true
    setError(null)
    Promise.all([dashboardApi.summary(year, month), dashboardApi.monthlyFlow(year, month, 6)])
      .then(([s, f]) => { if (active) { setSummary(s); setFlow(f) } })
      .catch((err) => { if (active) setError(err instanceof ApiError ? err.message : 'Erro ao carregar o dashboard.') })
    return () => { active = false }
  }, [year, month])

  const resultNegative = summary && Number(summary.monthly_result) < 0

  return (
    <div>
      <div className="page-header">
        <h1>Dashboard</h1>
      </div>

      <MonthPicker year={year} month={month} onChange={(y, m) => { setYear(y); setMonth(m) }} />

      {error && <ErrorState message={error} />}

      {summary === null ? <LoadingState /> : (
        <>
          <div className="grid grid--stats" style={{ marginBottom: '1rem' }}>
            <div className="stat-tile">
              <div className="stat-tile__label">Saldo disponível</div>
              <div className="stat-tile__value">{formatBRL(summary.accounts_balance)}</div>
            </div>
            <div className="stat-tile">
              <div className="stat-tile__label">Receitas do mês</div>
              <div className="stat-tile__value stat-tile__value--positive">{formatBRL(summary.monthly_income)}</div>
            </div>
            <div className="stat-tile">
              <div className="stat-tile__label">Despesas do mês</div>
              <div className="stat-tile__value stat-tile__value--negative">{formatBRL(summary.monthly_expense)}</div>
            </div>
            <div className="stat-tile">
              <div className="stat-tile__label">Resultado do mês</div>
              <div className={`stat-tile__value ${resultNegative ? 'stat-tile__value--negative' : 'stat-tile__value--positive'}`}>{formatBRL(summary.monthly_result)}</div>
            </div>
            <div className="stat-tile">
              <div className="stat-tile__label">Patrimônio líquido</div>
              <div className="stat-tile__value">{formatBRL(summary.net_worth.net_worth)}</div>
            </div>
            <div className="stat-tile">
              <div className="stat-tile__label">Total investido / valor de mercado</div>
              <div className="stat-tile__value" style={{ fontSize: '1.1rem' }}>{formatBRL(summary.total_invested)} / {formatBRL(summary.total_market_value)}</div>
            </div>
          </div>

          <div className="grid grid--two">
            <div className="card">
              <h3>Evolução mensal (receitas x despesas)</h3>
              {flow && flow.length > 0 ? (
                <GroupedBarChart
                  groups={flow.map((p) => ({
                    label: `${monthLabel(p.month).slice(0, 3)}`,
                    series: [
                      { label: 'Receita', value: Number(p.total_income), color: 'var(--series-increase)' },
                      { label: 'Despesa', value: Number(p.total_expense), color: 'var(--series-decrease)' },
                    ],
                  }))}
                />
              ) : <p>Sem dados suficientes.</p>}
            </div>

            <div className="card">
              <h3>Últimas movimentações</h3>
              {summary.recent_transactions.length === 0 ? <p>Nenhuma movimentação registrada.</p> : (
                <table>
                  <thead><tr><th>Data</th><th>Descrição</th><th>Tipo</th><th>Valor</th></tr></thead>
                  <tbody>
                    {summary.recent_transactions.map((t) => (
                      <tr key={t.id}>
                        <td>{formatDateBR(t.occurred_on)}</td>
                        <td>{t.description}</td>
                        <td><span className={`badge badge--${t.status}`} title={TRANSACTION_STATUS_LABEL[t.status]}>{TRANSACTION_TYPE_LABEL[t.type]}</span></td>
                        <td>{formatBRL(t.amount)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  )
}
