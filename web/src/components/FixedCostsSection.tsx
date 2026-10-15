import { useState } from 'react'
import { recurringApi } from '../api/endpoints'
import { ApiError } from '../api/client'
import type { Category, RecurringRule } from '../api/types'
import { formatBRL, monthLabel, MONTH_NAMES_PT, PERIODICITY_LABEL, todayISO } from '../utils/format'
import { Modal } from './Modal'

// Custos fixos de vida (regras de recorrência mensais e anuais). Lançar um
// custo no mês cria um lançamento previsto/pendente — nunca uma despesa paga.

function dueText(r: RecurringRule): string {
  const day = r.due_day ?? 1
  if (r.periodicity === 'yearly' && r.due_month) return `Anual · ${String(day).padStart(2, '0')}/${String(r.due_month).padStart(2, '0')}`
  if (r.periodicity === 'monthly') return `Mensal · dia ${day}`
  return PERIODICITY_LABEL[r.periodicity] ?? r.periodicity
}

export function FixedCostsSection({
  rules, year, month, categories, onChanged, onOccurrenceGenerated,
}: {
  rules: RecurringRule[] | null
  year: number
  month: number
  categories: Category[]
  onChanged: () => void
  onOccurrenceGenerated: () => void
}) {
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [editing, setEditing] = useState<RecurringRule | 'new' | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const categoryName = (id: string | null) => categories.find((c) => c.id === id)?.name ?? 'Sem categoria'

  async function handleGenerate(rule: RecurringRule) {
    setBusyId(rule.id)
    setError(null)
    setNotice(null)
    try {
      const res = await recurringApi.generate(rule.id, year, month)
      setNotice(res.created
        ? `"${rule.description}" lançado em ${monthLabel(month)}/${year} como ${res.status === 'planned' ? 'planejado' : 'pendente'}.`
        : `"${rule.description}" já estava lançado em ${monthLabel(month)}/${year}.`)
      onOccurrenceGenerated()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao lançar o custo no mês.')
    } finally {
      setBusyId(null)
    }
  }

  async function handleToggle(rule: RecurringRule) {
    setBusyId(rule.id)
    setError(null)
    try {
      await recurringApi.update(rule.id, {
        description: rule.description, planned_amount: rule.planned_amount, account_id: rule.account_id,
        category_id: rule.category_id, periodicity: rule.periodicity, start_date: rule.start_date,
        end_date: rule.end_date, due_day: rule.due_day, due_month: rule.due_month, active: !rule.active,
      })
      onChanged()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao atualizar o custo.')
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div className="card">
      <div className="page-header" style={{ marginBottom: '0.5rem' }}>
        <h3>Custos fixos</h3>
        <button type="button" className="btn btn--sm" onClick={() => setEditing('new')}>
          + Novo custo fixo
        </button>
      </div>
      <p className="stat-tile__hint" style={{ marginTop: 0, marginBottom: '0.75rem' }}>
        Contas que se repetem todo mês (aluguel, energia, internet) ou uma vez por ano (IPVA, IPTU, seguro).
        "Lançar" cria o lançamento previsto no mês selecionado em "Despesas do mês"; ele só conta como gasto depois de marcado como pago.
      </p>
      {error && <div className="alert alert--error">{error}</div>}
      {notice && <div className="alert alert--success">{notice}</div>}

      {rules === null ? null : rules.length === 0 ? (
        <p className="hbar__empty">Nenhum custo fixo cadastrado. Comece pelos principais, como aluguel, condomínio e contas de consumo.</p>
      ) : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Custo</th><th>Categoria</th><th>Vencimento</th>
                <th>Valor</th><th>Por mês</th><th>Por ano</th><th>Status</th><th></th>
              </tr>
            </thead>
            <tbody>
              {rules.map((r) => {
                const dueThisMonth = r.periodicity === 'monthly' || (r.periodicity === 'yearly' && r.due_month === month)
                const canGenerate = r.active && dueThisMonth && r.periodicity !== 'weekly'
                return (
                  <tr key={r.id}>
                    <td>{r.description}</td>
                    <td>{categoryName(r.category_id)}</td>
                    <td style={{ whiteSpace: 'nowrap' }}>{dueText(r)}</td>
                    <td>{formatBRL(r.planned_amount)}</td>
                    <td>{formatBRL(r.monthly_equivalent)}</td>
                    <td>{formatBRL(r.annual_cost)}</td>
                    <td><span className={`badge ${r.active ? 'badge--paid' : 'badge--neutral'}`}>{r.active ? 'Ativo' : 'Inativo'}</span></td>
                    <td style={{ whiteSpace: 'nowrap' }}>
                      <button
                        type="button"
                        className="btn btn--sm"
                        disabled={!canGenerate || busyId === r.id}
                        title={!dueThisMonth && r.due_month
                          ? `Vence em ${monthLabel(r.due_month).toLowerCase()}`
                          : `Lançar em ${monthLabel(month).toLowerCase()} de ${year}`}
                        onClick={() => handleGenerate(r)}
                      >
                        Lançar
                      </button>{' '}
                      <button type="button" className="btn btn--secondary btn--sm" onClick={() => setEditing(r)}>Editar</button>{' '}
                      <button type="button" className="btn btn--secondary btn--sm" disabled={busyId === r.id} onClick={() => handleToggle(r)}>
                        {r.active ? 'Desativar' : 'Ativar'}
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {editing && (
        <FixedCostFormModal
          existing={editing === 'new' ? null : editing}
          categories={categories.filter((c) => c.kind === 'expense')}
          onClose={() => setEditing(null)}
          onSaved={() => { setEditing(null); onChanged() }}
        />
      )}
    </div>
  )
}

function FixedCostFormModal({
  existing, categories, onClose, onSaved,
}: {
  existing: RecurringRule | null
  categories: Category[]
  onClose: () => void
  onSaved: () => void
}) {
  const [description, setDescription] = useState(existing?.description ?? '')
  const [plannedAmount, setPlannedAmount] = useState(existing?.planned_amount ?? '')
  const [periodicity, setPeriodicity] = useState<RecurringRule['periodicity']>(existing?.periodicity ?? 'monthly')
  const [dueDay, setDueDay] = useState(String(existing?.due_day ?? 5))
  const [dueMonth, setDueMonth] = useState(String(existing?.due_month ?? new Date().getMonth() + 1))
  const [categoryId, setCategoryId] = useState(existing?.category_id ?? categories[0]?.id ?? '')
  const [startDate, setStartDate] = useState(existing?.start_date ?? todayISO())
  const [endDate, setEndDate] = useState(existing?.end_date ?? '')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  async function handleSubmit() {
    setSaving(true)
    setFieldErrors({})
    setError(null)
    const body = {
      description, planned_amount: plannedAmount, periodicity,
      // Sem conta, a ocorrência é lançada na conta padrão (Carteira).
      account_id: existing?.account_id ?? null, category_id: categoryId || null,
      due_day: dueDay ? Number(dueDay) : null,
      due_month: periodicity === 'yearly' ? Number(dueMonth) : null,
      start_date: startDate, end_date: endDate || null,
      active: existing?.active ?? true,
    }
    try {
      if (existing) await recurringApi.update(existing.id, body)
      else await recurringApi.create(body)
      onSaved()
    } catch (err) {
      if (err instanceof ApiError && err.fields) setFieldErrors(err.fields)
      setError(err instanceof ApiError ? err.message : 'Erro ao salvar o custo fixo.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={existing ? 'Editar custo fixo' : 'Novo custo fixo'} onClose={onClose}>
      {error && <div className="alert alert--error">{error}</div>}
      {existing && (
        <p className="stat-tile__hint" style={{ marginTop: 0 }}>
          Alterações valem para os próximos lançamentos; os já lançados não são modificados.
        </p>
      )}
      <div className="form-grid">
        <div className="field">
          <label htmlFor="fc-desc">Descrição</label>
          <input id="fc-desc" value={description} onChange={(e) => setDescription(e.target.value)} />
          {fieldErrors.description && <span className="field-error">{fieldErrors.description}</span>}
        </div>
        <div className="field">
          <label htmlFor="fc-amount">Valor (R$)</label>
          <input id="fc-amount" type="number" step="0.01" min="0.01" value={plannedAmount} onChange={(e) => setPlannedAmount(e.target.value)} />
          {fieldErrors.planned_amount && <span className="field-error">{fieldErrors.planned_amount}</span>}
        </div>
        <div className="field">
          <label htmlFor="fc-period">Periodicidade</label>
          <select id="fc-period" value={periodicity} onChange={(e) => setPeriodicity(e.target.value as RecurringRule['periodicity'])}>
            <option value="monthly">Mensal</option>
            <option value="yearly">Anual</option>
            {existing?.periodicity === 'weekly' && <option value="weekly">Semanal</option>}
          </select>
          {fieldErrors.periodicity && <span className="field-error">{fieldErrors.periodicity}</span>}
        </div>
        <div className="field">
          <label htmlFor="fc-day">Dia de vencimento</label>
          <input id="fc-day" type="number" min="1" max="31" value={dueDay} onChange={(e) => setDueDay(e.target.value)} />
          {fieldErrors.due_day && <span className="field-error">{fieldErrors.due_day}</span>}
        </div>
        {periodicity === 'yearly' && (
          <div className="field">
            <label htmlFor="fc-month">Mês de vencimento</label>
            <select id="fc-month" value={dueMonth} onChange={(e) => setDueMonth(e.target.value)}>
              {MONTH_NAMES_PT.map((name, i) => <option key={name} value={i + 1}>{name}</option>)}
            </select>
            {fieldErrors.due_month && <span className="field-error">{fieldErrors.due_month}</span>}
          </div>
        )}
        <div className="field">
          <label htmlFor="fc-cat">Categoria</label>
          <select id="fc-cat" value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
            {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
          {fieldErrors.category_id && <span className="field-error">{fieldErrors.category_id}</span>}
        </div>
        <div className="field">
          <label htmlFor="fc-start">Início</label>
          <input id="fc-start" type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="fc-end">Término (opcional)</label>
          <input id="fc-end" type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
        </div>
      </div>
      <div className="modal__actions">
        <button type="button" className="btn btn--secondary" onClick={onClose}>Cancelar</button>
        <button type="button" className="btn" disabled={saving} onClick={handleSubmit}>{saving ? 'Salvando...' : 'Salvar'}</button>
      </div>
    </Modal>
  )
}
