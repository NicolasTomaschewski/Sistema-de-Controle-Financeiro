import { useEffect, useState } from 'react'
import { investmentsApi } from '../api/endpoints'
import { ApiError } from '../api/client'
import type { Investment, InvestmentMovement, Valuation } from '../api/types'
import { formatBRL, formatDateBR, INVESTMENT_TYPE_LABEL, todayISO } from '../utils/format'
import { LoadingState } from './States'
import { Modal } from './Modal'

// Modais de gestão de investimentos, usados na página Patrimônio: cadastro,
// movimentações (aporte, resgate, rendimento, taxa) e detalhes com histórico
// e atualização do valor de mercado.

const INVESTMENT_TYPES = Object.keys(INVESTMENT_TYPE_LABEL)

const MOVEMENT_TYPE_LABEL: Record<string, string> = {
  contribution: 'Aporte / compra',
  redemption: 'Resgate / venda',
  income: 'Rendimento',
  fee: 'Taxa',
}

export function CreateInvestmentModal({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const [name, setName] = useState('')
  const [assetType, setAssetType] = useState(INVESTMENT_TYPES[0])
  const [ticker, setTicker] = useState('')
  const [investedAmount, setInvestedAmount] = useState('0.00')
  const [marketValue, setMarketValue] = useState('0.00')
  const [positionDate, setPositionDate] = useState(todayISO())
  const [includeInNetWorth, setIncludeInNetWorth] = useState(true)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  async function handleSubmit() {
    setSaving(true)
    setFieldErrors({})
    try {
      await investmentsApi.create({
        name, asset_type: assetType, ticker: ticker || null, invested_amount: investedAmount,
        market_value: marketValue, position_date: positionDate, include_in_net_worth: includeInNetWorth,
      })
      onSaved()
    } catch (err) {
      if (err instanceof ApiError && err.fields) setFieldErrors(err.fields)
      setError(err instanceof ApiError ? err.message : 'Erro ao criar investimento.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title="Novo investimento" onClose={onClose}>
      {error && <div className="alert alert--error">{error}</div>}
      <div className="form-grid">
        <div className="field">
          <label>Nome</label>
          <input value={name} onChange={(e) => setName(e.target.value)} />
          {fieldErrors.name && <span className="field-error">{fieldErrors.name}</span>}
        </div>
        <div className="field">
          <label>Tipo de ativo</label>
          <select value={assetType} onChange={(e) => setAssetType(e.target.value)}>
            {INVESTMENT_TYPES.map((t) => <option key={t} value={t}>{INVESTMENT_TYPE_LABEL[t] ?? t}</option>)}
          </select>
        </div>
        <div className="field">
          <label>Ticker / código (opcional)</label>
          <input value={ticker} onChange={(e) => setTicker(e.target.value)} />
        </div>
        <div className="field">
          <label>Valor já investido (R$)</label>
          <input type="number" step="0.01" min="0" value={investedAmount} onChange={(e) => setInvestedAmount(e.target.value)} />
        </div>
        <div className="field">
          <label>Valor de mercado atual (R$)</label>
          <input type="number" step="0.01" min="0" value={marketValue} onChange={(e) => setMarketValue(e.target.value)} />
        </div>
        <div className="field">
          <label>Data de referência</label>
          <input type="date" value={positionDate} onChange={(e) => setPositionDate(e.target.value)} />
        </div>
        <div className="field" style={{ gridColumn: '1 / -1' }}>
          <label style={{ display: 'flex', alignItems: 'center', gap: '0.4rem' }}>
            <input type="checkbox" checked={includeInNetWorth} onChange={(e) => setIncludeInNetWorth(e.target.checked)} />
            Incluir no patrimônio
          </label>
          <span style={{ fontSize: '0.78rem', color: 'var(--color-text-muted)' }}>
            Desmarque se este valor não deve entrar no seu patrimônio (por exemplo, se já estiver contado em outro lugar).
          </span>
        </div>
      </div>
      <div className="modal__actions">
        <button type="button" className="btn btn--secondary" onClick={onClose}>Cancelar</button>
        <button type="button" className="btn" disabled={saving} onClick={handleSubmit}>{saving ? 'Salvando...' : 'Criar'}</button>
      </div>
    </Modal>
  )
}

export function MovementModal({
  investment, onClose, onSaved,
}: {
  investment: Investment
  onClose: () => void
  onSaved: () => void
}) {
  const [type, setType] = useState<'contribution' | 'redemption' | 'income' | 'fee'>('contribution')
  const [amount, setAmount] = useState('')
  const [quantity, setQuantity] = useState('')
  const [fees, setFees] = useState('0.00')
  const [occurredOn, setOccurredOn] = useState(todayISO())
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  async function handleSubmit() {
    setSaving(true)
    setError(null)
    try {
      await investmentsApi.recordMovement(investment.id, {
        type, amount, quantity: quantity || null, fees, occurred_on: occurredOn,
      })
      onSaved()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao registrar movimentação.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={`Movimentar — ${investment.name}`} onClose={onClose}>
      {error && <div className="alert alert--error">{error}</div>}
      <div className="form-grid">
        <div className="field">
          <label>Tipo</label>
          <select value={type} onChange={(e) => setType(e.target.value as typeof type)}>
            <option value="contribution">Aporte / compra</option>
            <option value="redemption">Resgate / venda</option>
            <option value="income">Rendimento recebido</option>
            <option value="fee">Taxa paga</option>
          </select>
        </div>
        <div className="field">
          <label>Valor (R$)</label>
          <input type="number" step="0.01" min="0.01" value={amount} onChange={(e) => setAmount(e.target.value)} />
        </div>
        {(type === 'contribution' || type === 'redemption') && (
          <div className="field">
            <label>Quantidade (opcional, para ativos com cotas)</label>
            <input type="number" step="0.00000001" min="0" value={quantity} onChange={(e) => setQuantity(e.target.value)} />
          </div>
        )}
        <div className="field">
          <label>Taxas (R$, opcional)</label>
          <input type="number" step="0.01" min="0" value={fees} onChange={(e) => setFees(e.target.value)} />
        </div>
        <div className="field">
          <label>Data</label>
          <input type="date" value={occurredOn} onChange={(e) => setOccurredOn(e.target.value)} />
        </div>
      </div>
      <div className="modal__actions">
        <button type="button" className="btn btn--secondary" onClick={onClose}>Cancelar</button>
        <button type="button" className="btn" disabled={saving} onClick={handleSubmit}>{saving ? 'Salvando...' : 'Registrar'}</button>
      </div>
    </Modal>
  )
}

export function DetailsModal({ investment, onClose }: { investment: Investment; onClose: () => void }) {
  const [movements, setMovements] = useState<InvestmentMovement[] | null>(null)
  const [valuations, setValuations] = useState<Valuation[] | null>(null)
  const [newValue, setNewValue] = useState('')
  const [newDate, setNewDate] = useState(todayISO())
  const [error, setError] = useState<string | null>(null)

  async function load() {
    try {
      const [m, v] = await Promise.all([investmentsApi.listMovements(investment.id), investmentsApi.listValuations(investment.id)])
      setMovements(m)
      setValuations(v)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao carregar detalhes.')
    }
  }

  useEffect(() => { void load() }, [investment.id])

  async function handleAddValuation() {
    try {
      await investmentsApi.addValuation(investment.id, { market_value: newValue, position_date: newDate })
      setNewValue('')
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao registrar valor de mercado.')
    }
  }

  return (
    <Modal title={`Detalhes — ${investment.name}`} onClose={onClose}>
      {error && <div className="alert alert--error">{error}</div>}

      <h4>Atualizar valor de mercado</h4>
      <div className="toolbar">
        <input type="number" step="0.01" min="0" placeholder="Novo valor" value={newValue} onChange={(e) => setNewValue(e.target.value)} style={{ maxWidth: '140px' }} />
        <input type="date" value={newDate} onChange={(e) => setNewDate(e.target.value)} />
        <button type="button" className="btn btn--sm" onClick={handleAddValuation} disabled={!newValue}>Registrar</button>
      </div>

      <h4>Histórico de valor de mercado</h4>
      {valuations === null ? <LoadingState /> : valuations.length === 0 ? <p>Nenhum registro.</p> : (
        <table>
          <thead><tr><th>Data</th><th>Valor</th></tr></thead>
          <tbody>
            {valuations.map((v) => <tr key={v.id}><td>{formatDateBR(v.position_date)}</td><td>{formatBRL(v.market_value)}</td></tr>)}
          </tbody>
        </table>
      )}

      <h4 style={{ marginTop: '1rem' }}>Movimentações</h4>
      {movements === null ? <LoadingState /> : movements.length === 0 ? <p>Nenhuma movimentação.</p> : (
        <table>
          <thead><tr><th>Data</th><th>Tipo</th><th>Valor</th><th>Ganho realizado</th></tr></thead>
          <tbody>
            {movements.map((m) => (
              <tr key={m.id}>
                <td>{formatDateBR(m.occurred_on)}</td>
                <td>{MOVEMENT_TYPE_LABEL[m.type] ?? m.type}</td>
                <td>{formatBRL(m.amount)}</td>
                <td>{m.realized_gain ? formatBRL(m.realized_gain) : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <div className="modal__actions">
        <button type="button" className="btn btn--secondary" onClick={onClose}>Fechar</button>
      </div>
    </Modal>
  )
}
