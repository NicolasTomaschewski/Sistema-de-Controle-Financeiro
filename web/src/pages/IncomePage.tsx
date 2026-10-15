import { useEffect, useState } from 'react'
import { categoriesApi, transactionsApi } from '../api/endpoints'
import { ApiError } from '../api/client'
import type { AnnualSummary, Category, CategoryBreakdown, MonthlySummary, Transaction } from '../api/types'
import { formatBRL, monthLabel } from '../utils/format'
import { MonthPicker } from '../components/MonthPicker'
import { LoadingState, ErrorState } from '../components/States'
import { TransactionsTable } from '../components/TransactionsTable'
import { TransactionFormModal } from '../components/TransactionFormModal'
import { HBarList } from '../components/HBarList'
import { ConfirmDialog } from '../components/Modal'

// Receitas = tudo o que o usuário ganha. Os totais consideram apenas receitas
// recebidas (status pago); previstas e pendentes aparecem na lista, mas não somam.

function sourcesToBars(items: CategoryBreakdown[]) {
  return items.map((c) => ({
    key: c.category_id ?? 'none',
    label: c.category_name,
    value: Number(c.total),
    valueText: formatBRL(c.total),
  }))
}

export default function IncomePage() {
  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)

  const [categories, setCategories] = useState<Category[]>([])
  const [transactions, setTransactions] = useState<Transaction[] | null>(null)
  const [summary, setSummary] = useState<MonthlySummary | null>(null)
  const [annual, setAnnual] = useState<AnnualSummary | null>(null)
  const [error, setError] = useState<string | null>(null)

  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<Transaction | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Transaction | null>(null)

  async function loadStatic() {
    try {
      setCategories(await categoriesApi.list())
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao carregar dados.')
    }
  }

  async function loadPeriod() {
    try {
      const from = `${year}-${String(month).padStart(2, '0')}-01`
      const lastDay = new Date(year, month, 0).getDate()
      const to = `${year}-${String(month).padStart(2, '0')}-${String(lastDay).padStart(2, '0')}`

      const [{ data }, monthSummary, yearSummary] = await Promise.all([
        transactionsApi.list({ type: 'income', from, to, limit: 200 }),
        transactionsApi.monthlySummary(year, month, true),
        transactionsApi.annualSummary(year),
      ])
      setTransactions(data)
      setSummary(monthSummary)
      setAnnual(yearSummary)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao carregar receitas.')
    }
  }

  useEffect(() => { void loadStatic() }, [])
  useEffect(() => { void loadPeriod() }, [year, month])

  function refresh() {
    setCreating(false)
    setEditing(null)
    void loadPeriod()
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await transactionsApi.remove(deleteTarget.id)
      setDeleteTarget(null)
      void loadPeriod()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao excluir lançamento.')
    }
  }

  const incomeCategories = categories.filter((c) => c.kind === 'income')
  const monthName = monthLabel(month).toLowerCase()

  return (
    <div>
      <div className="page-header">
        <h1>Receitas</h1>
        <button type="button" className="btn" onClick={() => setCreating(true)}>+ Nova receita</button>
      </div>
      <p className="page-intro">Tudo o que você ganha: salário, trabalhos, rendimentos, aluguéis e outras entradas.</p>

      <MonthPicker year={year} month={month} onChange={(y, m) => { setYear(y); setMonth(m) }} />

      {error && <ErrorState message={error} />}

      {summary && annual && (
        <div className="grid grid--stats" style={{ marginBottom: '1rem' }}>
          <div className="stat-tile">
            <div className="stat-tile__label">Recebido em {monthName}</div>
            <div className="stat-tile__value stat-tile__value--positive">{formatBRL(summary.total_income)}</div>
          </div>
          {summary.previous_month && (
            <div className="stat-tile">
              <div className="stat-tile__label">Recebido no mês anterior</div>
              <div className="stat-tile__value">{formatBRL(summary.previous_month.total_income)}</div>
            </div>
          )}
          <div className="stat-tile">
            <div className="stat-tile__label">Recebido em {year}</div>
            <div className="stat-tile__value">{formatBRL(annual.total_income)}</div>
            <div className="stat-tile__hint">De 1º de janeiro a 31 de dezembro</div>
          </div>
        </div>
      )}

      {summary && annual && (
        <div className="grid grid--two" style={{ marginBottom: '1rem' }}>
          <div className="card">
            <h3>Fontes de renda em {monthName}</h3>
            <HBarList
              tone="increase"
              ariaLabel={`Receitas recebidas por fonte em ${monthName} de ${year}`}
              emptyText="Nenhuma receita recebida neste mês."
              items={sourcesToBars(summary.income_by_category)}
            />
          </div>
          <div className="card">
            <h3>Fontes de renda em {year}</h3>
            <HBarList
              tone="increase"
              ariaLabel={`Receitas recebidas por fonte em ${year}`}
              emptyText="Nenhuma receita recebida neste ano."
              items={sourcesToBars(annual.income_by_category)}
            />
          </div>
        </div>
      )}

      <h3>Lançamentos de {monthName}</h3>
      <p className="stat-tile__hint" style={{ marginTop: 0, marginBottom: '0.75rem' }}>
        Receitas previstas ou pendentes aparecem aqui, mas só entram nos totais depois de recebidas.
      </p>
      {transactions === null ? <LoadingState /> : (
        <TransactionsTable
          transactions={transactions}
          categories={categories}
          onEdit={setEditing}
          onDelete={setDeleteTarget}
        />
      )}

      {creating && (
        <TransactionFormModal type="income" categories={incomeCategories} onClose={() => setCreating(false)} onSaved={refresh} />
      )}
      {editing && (
        <TransactionFormModal type="income" categories={incomeCategories} existing={editing} onClose={() => setEditing(null)} onSaved={refresh} />
      )}
      {deleteTarget && (
        <ConfirmDialog
          title="Excluir receita"
          message={`Tem certeza que deseja excluir "${deleteTarget.description}"?`}
          confirmLabel="Excluir"
          danger
          onConfirm={handleDelete}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
