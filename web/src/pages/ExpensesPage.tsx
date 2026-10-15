import { useEffect, useState } from 'react'
import { categoriesApi, recurringApi, transactionsApi } from '../api/endpoints'
import { ApiError } from '../api/client'
import type { Category, CostSummary, MonthlySummary, RecurringRule, Transaction } from '../api/types'
import { formatBRL } from '../utils/format'
import { MonthPicker } from '../components/MonthPicker'
import { LoadingState, ErrorState } from '../components/States'
import { TransactionsTable } from '../components/TransactionsTable'
import { TransactionFormModal } from '../components/TransactionFormModal'
import { BudgetsSection } from '../components/BudgetsSection'
import { FixedCostsSection } from '../components/FixedCostsSection'
import { HBarList } from '../components/HBarList'
import { ConfirmDialog } from '../components/Modal'

// Despesas = custo de vida. A parte de cima trata dos custos fixos (mensais e
// anuais) e do custo de vida consolidado; a de baixo, das despesas efetivas
// de cada mês (lançamentos, orçamento e filtros).

export default function ExpensesPage() {
  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth() + 1)

  const [categories, setCategories] = useState<Category[]>([])
  const [rules, setRules] = useState<RecurringRule[] | null>(null)
  const [costs, setCosts] = useState<CostSummary | null>(null)
  const [transactions, setTransactions] = useState<Transaction[] | null>(null)
  const [summary, setSummary] = useState<MonthlySummary | null>(null)
  const [error, setError] = useState<string | null>(null)

  const [statusFilter, setStatusFilter] = useState('')
  const [categoryFilter, setCategoryFilter] = useState('')

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

  async function loadFixedCosts() {
    try {
      const [r, c] = await Promise.all([recurringApi.list(false), recurringApi.summary()])
      setRules([...r].sort((a, b) => Number(b.active) - Number(a.active)))
      setCosts(c)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao carregar custos fixos.')
    }
  }

  async function loadMonth() {
    try {
      const from = `${year}-${String(month).padStart(2, '0')}-01`
      const lastDay = new Date(year, month, 0).getDate()
      const to = `${year}-${String(month).padStart(2, '0')}-${String(lastDay).padStart(2, '0')}`

      const { data } = await transactionsApi.list({
        type: 'expense', from, to, status: statusFilter || undefined, category_id: categoryFilter || undefined, limit: 200,
      })
      setTransactions(data)
      setSummary(await transactionsApi.monthlySummary(year, month, true))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao carregar lançamentos.')
    }
  }

  useEffect(() => { void loadStatic(); void loadFixedCosts() }, [])
  useEffect(() => { void loadMonth() }, [year, month, statusFilter, categoryFilter])

  function refresh() {
    setCreating(false)
    setEditing(null)
    void loadMonth()
  }

  async function handleDelete() {
    if (!deleteTarget) return
    try {
      await transactionsApi.remove(deleteTarget.id)
      setDeleteTarget(null)
      void loadMonth()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao excluir lançamento.')
    }
  }

  const expenseCategories = categories.filter((c) => c.kind === 'expense')
  const categoryName = (id: string | null) => categories.find((c) => c.id === id)?.name ?? 'Sem categoria'

  return (
    <div>
      <div className="page-header">
        <h1>Despesas</h1>
        <button type="button" className="btn btn--secondary" onClick={() => setCreating(true)}>+ Despesa avulsa</button>
      </div>
      <p className="page-intro">Seu custo de vida: custos fixos mensais e anuais, e o que foi gasto em cada mês.</p>

      {error && <ErrorState message={error} />}

      {costs && (
        <div className="grid grid--stats" style={{ marginBottom: '1rem' }}>
          <div className="stat-tile">
            <div className="stat-tile__label">Custos mensais</div>
            <div className="stat-tile__value">{formatBRL(costs.monthly_rules_total)}</div>
            <div className="stat-tile__hint">Soma dos custos fixos que vencem todo mês</div>
          </div>
          <div className="stat-tile">
            <div className="stat-tile__label">Custos anuais</div>
            <div className="stat-tile__value">{formatBRL(costs.yearly_rules_total)}</div>
            <div className="stat-tile__hint">Pagos uma vez por ano (IPVA, IPTU, seguros...)</div>
          </div>
          <div className="stat-tile">
            <div className="stat-tile__label">Custo de vida por mês</div>
            <div className="stat-tile__value stat-tile__value--negative">{formatBRL(costs.monthly_equivalent)}</div>
            <div className="stat-tile__hint">Mensais + anuais divididos por 12</div>
          </div>
          <div className="stat-tile">
            <div className="stat-tile__label">Custo de vida por ano</div>
            <div className="stat-tile__value">{formatBRL(costs.annual_total)}</div>
            <div className="stat-tile__hint">Mensais × 12 + anuais</div>
          </div>
        </div>
      )}

      <FixedCostsSection
        rules={rules}
        year={year}
        month={month}
       
        categories={categories}
        onChanged={loadFixedCosts}
        onOccurrenceGenerated={loadMonth}
      />

      {costs && costs.by_category.length > 0 && (
        <div className="card">
          <h3>Custo de vida por categoria</h3>
          <HBarList
            tone="decrease"
            ariaLabel="Custo fixo anual por categoria"
            emptyText="Nenhum custo fixo ativo."
            items={costs.by_category.map((c) => ({
              key: c.category_id ?? 'none',
              label: categoryName(c.category_id),
              value: Number(c.annual_total),
              valueText: `${formatBRL(c.annual_total)}/ano`,
              detail: `${formatBRL(c.monthly_equivalent)}/mês`,
            }))}
          />
        </div>
      )}

      <h2 className="section-title">Despesas do mês</h2>
      <MonthPicker year={year} month={month} onChange={(y, m) => { setYear(y); setMonth(m) }} />

      {summary && (
        <div className="grid grid--stats" style={{ marginBottom: '1rem' }}>
          <div className="stat-tile">
            <div className="stat-tile__label">Pago no mês</div>
            <div className="stat-tile__value">{formatBRL(summary.total_expense)}</div>
            <div className="stat-tile__hint">Somente despesas com status pago</div>
          </div>
          {summary.previous_month && (
            <div className="stat-tile">
              <div className="stat-tile__label">Pago no mês anterior</div>
              <div className="stat-tile__value">{formatBRL(summary.previous_month.total_expense)}</div>
            </div>
          )}
        </div>
      )}

      <div style={{ marginBottom: '1rem' }}>
        <BudgetsSection year={year} month={month} categories={categories} />
      </div>

      <div className="toolbar">
        <div className="field" style={{ minWidth: '160px' }}>
          <label htmlFor="exp-cat">Categoria</label>
          <select id="exp-cat" value={categoryFilter} onChange={(e) => setCategoryFilter(e.target.value)}>
            <option value="">Todas</option>
            {expenseCategories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
        </div>
        <div className="field" style={{ minWidth: '160px' }}>
          <label htmlFor="exp-status">Status</label>
          <select id="exp-status" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
            <option value="">Todos</option>
            <option value="paid">Pagas</option>
            <option value="pending">Pendentes</option>
            <option value="planned">Planejadas</option>
            <option value="cancelled">Canceladas</option>
          </select>
        </div>
      </div>

      {transactions === null ? <LoadingState /> : (
        <TransactionsTable
          transactions={transactions}
          categories={categories}
          onEdit={setEditing}
          onDelete={setDeleteTarget}
        />
      )}

      {creating && (
        <TransactionFormModal type="expense" categories={expenseCategories} onClose={() => setCreating(false)} onSaved={refresh} />
      )}
      {editing && (
        <TransactionFormModal type="expense" categories={expenseCategories} existing={editing} onClose={() => setEditing(null)} onSaved={refresh} />
      )}
      {deleteTarget && (
        <ConfirmDialog
          title="Excluir despesa"
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
