import { useState } from 'react'
import { Modal } from './Modal'
import { transactionsApi } from '../api/endpoints'
import { ApiError } from '../api/client'
import type { Category, Transaction, TransactionStatus } from '../api/types'
import { todayISO } from '../utils/format'

type Props = {
  type: 'income' | 'expense'
  categories: Category[]
  existing?: Transaction | null
  onClose: () => void
  onSaved: () => void
}

export function TransactionFormModal({ type, categories, existing, onClose, onSaved }: Props) {
  const [categoryId, setCategoryId] = useState(existing?.category_id ?? categories[0]?.id ?? '')
  const [amount, setAmount] = useState(existing?.amount ?? '')
  const [description, setDescription] = useState(existing?.description ?? '')
  const [occurredOn, setOccurredOn] = useState(existing?.occurred_on ?? todayISO())
  const [status, setStatus] = useState<TransactionStatus>(existing?.status ?? 'paid')
  const [notes, setNotes] = useState(existing?.notes ?? '')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const isTransferLeg = existing?.transfer_id != null || existing?.investment_transaction_id != null

  async function handleSubmit() {
    setSaving(true)
    setError(null)
    setFieldErrors({})
    // Sem conta informada, o backend usa a conta padrão do usuário (Carteira);
    // na edição, o lançamento permanece na conta em que já estava.
    const body = {
      ...(existing ? { account_id: existing.account_id } : {}),
      category_id: categoryId || null,
      type,
      status,
      amount,
      description,
      occurred_on: occurredOn,
      notes: notes || null,
    }
    try {
      if (existing) {
        await transactionsApi.update(existing.id, body)
      } else {
        await transactionsApi.create(body)
      }
      onSaved()
    } catch (err) {
      if (err instanceof ApiError && err.fields) setFieldErrors(err.fields)
      setError(err instanceof ApiError ? err.message : 'Erro ao salvar lançamento.')
    } finally {
      setSaving(false)
    }
  }

  const title = existing ? (type === 'income' ? 'Editar receita' : 'Editar despesa') : (type === 'income' ? 'Nova receita' : 'Nova despesa')

  return (
    <Modal title={title} onClose={onClose}>
      {isTransferLeg && (
        <div className="alert alert--error">
          Este lançamento foi gerado por uma transferência e não pode ser editado individualmente.
        </div>
      )}
      {error && <div className="alert alert--error">{error}</div>}
      <div className="form-grid">
        <div className="field">
          <label>Descrição</label>
          <input value={description} onChange={(e) => setDescription(e.target.value)} disabled={isTransferLeg} />
          {fieldErrors.description && <span className="field-error">{fieldErrors.description}</span>}
        </div>
        <div className="field">
          <label>Valor (R$)</label>
          <input type="number" step="0.01" min="0.01" value={amount} onChange={(e) => setAmount(e.target.value)} disabled={isTransferLeg} />
          {fieldErrors.amount && <span className="field-error">{fieldErrors.amount}</span>}
        </div>
        <div className="field">
          <label>Categoria</label>
          <select value={categoryId} onChange={(e) => setCategoryId(e.target.value)} disabled={isTransferLeg}>
            <option value="">Sem categoria</option>
            {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
          {fieldErrors.category_id && <span className="field-error">{fieldErrors.category_id}</span>}
        </div>
        <div className="field">
          <label>Data</label>
          <input type="date" value={occurredOn} onChange={(e) => setOccurredOn(e.target.value)} disabled={isTransferLeg} />
        </div>
        <div className="field">
          <label>Status</label>
          <select value={status} onChange={(e) => setStatus(e.target.value as TransactionStatus)} disabled={isTransferLeg}>
            <option value="paid">{type === 'income' ? 'Recebida' : 'Paga'}</option>
            <option value="pending">Pendente</option>
            <option value="planned">Planejada</option>
            <option value="cancelled">Cancelada</option>
          </select>
        </div>
        <div className="field" style={{ gridColumn: '1 / -1' }}>
          <label>Observações (opcional)</label>
          <textarea rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} disabled={isTransferLeg} />
        </div>
      </div>
      <div className="modal__actions">
        <button type="button" className="btn btn--secondary" onClick={onClose}>Cancelar</button>
        {!isTransferLeg && (
          <button type="button" className="btn" disabled={saving} onClick={handleSubmit}>{saving ? 'Salvando...' : 'Salvar'}</button>
        )}
      </div>
    </Modal>
  )
}
