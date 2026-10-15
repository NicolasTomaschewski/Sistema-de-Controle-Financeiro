import { useEffect, useState } from 'react'
import { budgetsApi } from '../api/endpoints'
import { ApiError } from '../api/client'
import type { BudgetSummary, Category } from '../api/types'
import { formatBRL } from '../utils/format'
import { Modal } from './Modal'

export function BudgetsSection({ year, month, categories }: { year: number; month: number; categories: Category[] }) {
  const [summaries, setSummaries] = useState<BudgetSummary[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [categoryId, setCategoryId] = useState('')
  const [plannedAmount, setPlannedAmount] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)

  async function load() {
    try {
      setSummaries(await budgetsApi.summary(year, month))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao carregar orçamento.')
    }
  }

  useEffect(() => { void load() }, [year, month])

  async function handleCreate() {
    setSaving(true)
    setFieldErrors({})
    try {
      await budgetsApi.create({ category_id: categoryId, year, month, planned_amount: plannedAmount })
      setCreating(false)
      setPlannedAmount('')
      await load()
    } catch (err) {
      if (err instanceof ApiError && err.fields) setFieldErrors(err.fields)
      else setError(err instanceof ApiError ? err.message : 'Erro ao criar orçamento.')
    } finally {
      setSaving(false)
    }
  }

  const expenseCategories = categories.filter((c) => c.kind === 'expense')
  const usedCategoryIds = new Set(summaries?.map((s) => s.category_id))
  const availableCategories = expenseCategories.filter((c) => !usedCategoryIds.has(c.id))

  return (
    <div className="card">
      <div className="page-header" style={{ marginBottom: '0.5rem' }}>
        <h3>Orçamento do mês</h3>
        <button
          type="button"
          className="btn btn--secondary btn--sm"
          disabled={availableCategories.length === 0}
          onClick={() => { setCategoryId(availableCategories[0]?.id ?? ''); setPlannedAmount(''); setFieldErrors({}); setCreating(true) }}
        >
          + Definir limite
        </button>
      </div>
      {error && <div className="alert alert--error">{error}</div>}
      {summaries === null ? null : summaries.length === 0 ? (
        <p style={{ color: 'var(--color-text-muted)', fontSize: '0.88rem' }}>Nenhum orçamento definido para este mês.</p>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.9rem' }}>
          {summaries.map((s) => {
            const pct = Math.min(Number(s.percent_used), 100)
            return (
              <div key={s.category_id}>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem', marginBottom: '0.25rem' }}>
                  <strong>{s.category_name}</strong>
                  <span>{formatBRL(s.spent_amount)} de {formatBRL(s.planned_amount)} {s.over_limit && <span style={{ color: 'var(--color-danger)' }}>(excedido)</span>}</span>
                </div>
                <div className="progress-bar">
                  <div className={`progress-bar__fill ${s.over_limit ? 'progress-bar__fill--over' : ''}`} style={{ width: `${pct}%` }} />
                </div>
              </div>
            )
          })}
        </div>
      )}

      {creating && (
        <Modal title="Definir limite de orçamento" onClose={() => setCreating(false)}>
          <div className="form-grid">
            <div className="field">
              <label>Categoria</label>
              <select value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
                {availableCategories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
            <div className="field">
              <label>Valor planejado (R$)</label>
              <input type="number" step="0.01" min="0.01" value={plannedAmount} onChange={(e) => setPlannedAmount(e.target.value)} />
              {fieldErrors.planned_amount && <span className="field-error">{fieldErrors.planned_amount}</span>}
            </div>
          </div>
          <div className="modal__actions">
            <button type="button" className="btn btn--secondary" onClick={() => setCreating(false)}>Cancelar</button>
            <button type="button" className="btn" disabled={saving} onClick={handleCreate}>{saving ? 'Salvando...' : 'Salvar'}</button>
          </div>
        </Modal>
      )}
    </div>
  )
}
