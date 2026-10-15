import { useEffect, useState } from 'react'
import { assetsApi, investmentsApi, netWorthApi } from '../api/endpoints'
import { ApiError } from '../api/client'
import type { Asset, Investment, InvestmentSummary, NetWorthComposition, Valuation } from '../api/types'
import { ASSET_TYPE_LABEL, formatBRL, formatDateBR, formatDateTimeBR, formatPercent, INVESTMENT_TYPE_LABEL, todayISO } from '../utils/format'
import { CreateInvestmentModal, DetailsModal, MovementModal } from '../components/InvestmentModals'
import { ErrorState, LoadingState } from '../components/States'
import { Modal, ConfirmDialog } from '../components/Modal'
import { PieChart } from '../components/PieChart'
import { groupSmallSlices, type PieSlice } from '../utils/pie'

// Patrimônio = bens (imóveis, veículos...) + investimentos. É também onde os
// investimentos são cadastrados e movimentados. Dívidas e saldo de contas não
// entram nesta visão; o patrimônio líquido completo continua no Dashboard.

const ASSET_TYPES = ['imovel', 'veiculo', 'participacao_empresarial', 'dinheiro_em_especie', 'outro']

function sliceLabel(group: string, type: string): string {
  return group === 'asset' ? (ASSET_TYPE_LABEL[type] ?? type) : (INVESTMENT_TYPE_LABEL[type] ?? type)
}

function formatShare(value: number): string {
  return `${value.toLocaleString('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%`
}

export default function NetWorthPage() {
  const [composition, setComposition] = useState<NetWorthComposition | null>(null)
  const [assets, setAssets] = useState<Asset[] | null>(null)
  const [investments, setInvestments] = useState<Investment[] | null>(null)
  const [investmentSummary, setInvestmentSummary] = useState<InvestmentSummary | null>(null)
  const [error, setError] = useState<string | null>(null)

  const [creatingAsset, setCreatingAsset] = useState(false)
  const [valuationTarget, setValuationTarget] = useState<Asset | null>(null)
  const [historyTarget, setHistoryTarget] = useState<Asset | null>(null)
  const [deleteAssetTarget, setDeleteAssetTarget] = useState<Asset | null>(null)

  const [creatingInvestment, setCreatingInvestment] = useState(false)
  const [movementTarget, setMovementTarget] = useState<Investment | null>(null)
  const [detailsTarget, setDetailsTarget] = useState<Investment | null>(null)
  const [deleteInvestmentTarget, setDeleteInvestmentTarget] = useState<Investment | null>(null)

  async function loadAll() {
    try {
      const [c, a, inv, invSummary] = await Promise.all([
        netWorthApi.composition(), assetsApi.list(), investmentsApi.list(), investmentsApi.summary(),
      ])
      setComposition(c)
      setAssets(a)
      setInvestments(inv)
      setInvestmentSummary(invSummary)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao carregar o patrimônio.')
    }
  }

  useEffect(() => { void loadAll() }, [])

  async function handleDeleteAsset() {
    if (!deleteAssetTarget) return
    try {
      await assetsApi.remove(deleteAssetTarget.id)
      setDeleteAssetTarget(null)
      await loadAll()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao excluir bem.')
    }
  }

  async function handleDeleteInvestment() {
    if (!deleteInvestmentTarget) return
    try {
      await investmentsApi.remove(deleteInvestmentTarget.id)
      setDeleteInvestmentTarget(null)
      await loadAll()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao excluir investimento.')
    }
  }

  const slices: PieSlice[] = composition
    ? groupSmallSlices(
      composition.slices.map((sl) => ({
        key: `${sl.group}:${sl.type}`,
        label: sliceLabel(sl.group, sl.type),
        value: Number(sl.value),
        share: Number(sl.share_percent),
        valueText: formatBRL(sl.value),
        shareText: formatShare(Number(sl.share_percent)),
      })),
      formatBRL,
      formatShare,
    )
    : []

  return (
    <div>
      <div className="page-header">
        <h1>Patrimônio</h1>
        <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
          <button type="button" className="btn btn--secondary" onClick={() => setCreatingAsset(true)}>+ Novo bem</button>
          <button type="button" className="btn" onClick={() => setCreatingInvestment(true)}>+ Novo investimento</button>
        </div>
      </div>
      <p className="page-intro">Seus bens e investimentos. Dívidas não entram nesta visão.</p>

      {error && <ErrorState message={error} />}

      {composition === null ? <LoadingState /> : (
        <>
          <div className="grid grid--stats" style={{ marginBottom: '1rem' }}>
            <div className="stat-tile">
              <div className="stat-tile__label">Bens</div>
              <div className="stat-tile__value">{formatBRL(composition.assets_value)}</div>
              <div className="stat-tile__hint">Imóveis, veículos e outros bens</div>
            </div>
            <div className="stat-tile">
              <div className="stat-tile__label">Investimentos</div>
              <div className="stat-tile__value">{formatBRL(composition.investments_value)}</div>
              <div className="stat-tile__hint">Valor de mercado informado</div>
            </div>
            <div className="stat-tile">
              <div className="stat-tile__label">Patrimônio total</div>
              <div className="stat-tile__value stat-tile__value--positive">{formatBRL(composition.total)}</div>
              <div className="stat-tile__hint">Bens + investimentos</div>
            </div>
          </div>

          <div className="card" style={{ marginBottom: '1rem' }}>
            <h3>Composição do patrimônio</h3>
            <p className="stat-tile__hint" style={{ marginTop: 0, marginBottom: '1rem' }}>
              Calculado em {formatDateTimeBR(composition.as_of)}. Os valores são estimativas informadas por você, não cotações de mercado.
            </p>
            {slices.length === 0 ? (
              <p className="hbar__empty">Cadastre um bem ou um investimento para ver a composição do seu patrimônio.</p>
            ) : (
              <PieChart
                slices={slices}
                totalLabel="Total"
                totalText={formatBRL(composition.total)}
                ariaLabel={`Composição do patrimônio: ${slices.map((s) => `${s.label} ${s.shareText}`).join(', ')}`}
              />
            )}
          </div>
        </>
      )}

      <div className="grid grid--two">
        <div className="card">
          <div className="page-header" style={{ marginBottom: '0.5rem' }}>
            <h3>Bens</h3>
            <button type="button" className="btn btn--secondary btn--sm" onClick={() => setCreatingAsset(true)}>+ Novo bem</button>
          </div>
          {assets === null ? null : assets.length === 0 ? (
            <p className="hbar__empty">Nenhum bem cadastrado. Adicione imóveis, veículos ou outros bens.</p>
          ) : (
            <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
              {assets.map((a) => (
                <li key={a.id} style={{ padding: '0.5rem 0', borderBottom: '1px solid var(--color-border)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', gap: '0.75rem' }}>
                    <strong>{a.name}</strong>
                    <span>{formatBRL(a.current_value)}</span>
                  </div>
                  <div style={{ fontSize: '0.78rem', color: 'var(--color-text-muted)' }}>
                    {ASSET_TYPE_LABEL[a.type] ?? a.type} · estimativa em {formatDateBR(a.valuation_date)}
                  </div>
                  <div style={{ marginTop: '0.3rem' }}>
                    <button type="button" className="btn btn--secondary btn--sm" onClick={() => setValuationTarget(a)}>Atualizar valor</button>{' '}
                    <button type="button" className="btn btn--secondary btn--sm" onClick={() => setHistoryTarget(a)}>Histórico</button>{' '}
                    <button type="button" className="btn btn--danger btn--sm" onClick={() => setDeleteAssetTarget(a)}>Excluir</button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="card">
          <div className="page-header" style={{ marginBottom: '0.5rem' }}>
            <h3>Investimentos</h3>
            <button type="button" className="btn btn--secondary btn--sm" onClick={() => setCreatingInvestment(true)}>+ Novo investimento</button>
          </div>
          {investmentSummary && investments && investments.length > 0 && (
            <dl className="mini-stats">
              <div><dt>Total aportado</dt><dd>{formatBRL(investmentSummary.total_invested)}</dd></div>
              <div><dt>Valor de mercado</dt><dd>{formatBRL(investmentSummary.total_market_value)}</dd></div>
              <div>
                <dt>Ganho não realizado</dt>
                <dd className={Number(investmentSummary.unrealized_gain) >= 0 ? 'stat-tile__value--positive' : 'stat-tile__value--negative'}>
                  {formatBRL(investmentSummary.unrealized_gain)}
                </dd>
              </div>
              <div>
                <dt>Resultado realizado</dt>
                <dd className={Number(investmentSummary.total_realized_result) >= 0 ? 'stat-tile__value--positive' : 'stat-tile__value--negative'}>
                  {formatBRL(investmentSummary.total_realized_result)}
                </dd>
              </div>
            </dl>
          )}
          {investments === null ? null : investments.length === 0 ? (
            <p className="hbar__empty">Nenhum investimento cadastrado. Adicione renda fixa, ações, fundos, previdência ou outros.</p>
          ) : (
            <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
              {investments.map((inv) => (
                <li key={inv.id} style={{ padding: '0.5rem 0', borderBottom: '1px solid var(--color-border)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', gap: '0.75rem' }}>
                    <strong>{inv.name}{inv.ticker && <span style={{ color: 'var(--color-text-muted)', fontWeight: 400 }}> ({inv.ticker})</span>}</strong>
                    <span>{formatBRL(inv.market_value)}</span>
                  </div>
                  <div style={{ fontSize: '0.78rem', color: 'var(--color-text-muted)' }}>
                    {INVESTMENT_TYPE_LABEL[inv.asset_type] ?? inv.asset_type} · aportado {formatBRL(inv.invested_amount)}
                    {inv.simple_return_percent !== null && <> · rentabilidade {formatPercent(inv.simple_return_percent)}</>}
                    {' '}· posição em {formatDateBR(inv.position_date)}
                    {!inv.include_in_net_worth && <> · <span className="badge badge--neutral">fora do patrimônio</span></>}
                  </div>
                  <div style={{ marginTop: '0.3rem' }}>
                    <button type="button" className="btn btn--secondary btn--sm" onClick={() => setMovementTarget(inv)}>Aporte / resgate</button>{' '}
                    <button type="button" className="btn btn--secondary btn--sm" onClick={() => setDetailsTarget(inv)}>Atualizar valor e histórico</button>{' '}
                    <button type="button" className="btn btn--danger btn--sm" onClick={() => setDeleteInvestmentTarget(inv)}>Excluir</button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {creatingAsset && <AssetFormModal onClose={() => setCreatingAsset(false)} onSaved={() => { setCreatingAsset(false); void loadAll() }} />}
      {valuationTarget && (
        <ValuationFormModal
          item={valuationTarget}
          onClose={() => setValuationTarget(null)}
          onSaved={() => { setValuationTarget(null); void loadAll() }}
        />
      )}
      {historyTarget && <HistoryModal item={historyTarget} onClose={() => setHistoryTarget(null)} />}

      {creatingInvestment && (
        <CreateInvestmentModal onClose={() => setCreatingInvestment(false)} onSaved={() => { setCreatingInvestment(false); void loadAll() }} />
      )}
      {movementTarget && (
        <MovementModal investment={movementTarget} onClose={() => setMovementTarget(null)} onSaved={() => { setMovementTarget(null); void loadAll() }} />
      )}
      {detailsTarget && <DetailsModal investment={detailsTarget} onClose={() => { setDetailsTarget(null); void loadAll() }} />}
      {deleteInvestmentTarget && (
        <ConfirmDialog
          title="Excluir investimento"
          message={`Excluir "${deleteInvestmentTarget.name}"? Se houver movimentações registradas, o investimento será encerrado em vez de excluído, preservando o histórico.`}
          confirmLabel="Excluir"
          danger
          onConfirm={handleDeleteInvestment}
          onCancel={() => setDeleteInvestmentTarget(null)}
        />
      )}
      {deleteAssetTarget && (
        <ConfirmDialog title="Excluir bem" message={`Excluir "${deleteAssetTarget.name}"?`} confirmLabel="Excluir" danger onConfirm={handleDeleteAsset} onCancel={() => setDeleteAssetTarget(null)} />
      )}
    </div>
  )
}

function AssetFormModal({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const [name, setName] = useState('')
  const [type, setType] = useState(ASSET_TYPES[0])
  const [currentValue, setCurrentValue] = useState('')
  const [valuationDate, setValuationDate] = useState(todayISO())
  const [notes, setNotes] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit() {
    setSaving(true)
    setFieldErrors({})
    try {
      await assetsApi.create({ name, type, current_value: currentValue, valuation_date: valuationDate, notes: notes || null })
      onSaved()
    } catch (err) {
      if (err instanceof ApiError && err.fields) setFieldErrors(err.fields)
      setError(err instanceof ApiError ? err.message : 'Erro ao criar bem.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title="Novo bem" onClose={onClose}>
      {error && <div className="alert alert--error">{error}</div>}
      <div className="form-grid">
        <div className="field">
          <label>Nome</label>
          <input value={name} onChange={(e) => setName(e.target.value)} />
          {fieldErrors.name && <span className="field-error">{fieldErrors.name}</span>}
        </div>
        <div className="field">
          <label>Tipo</label>
          <select value={type} onChange={(e) => setType(e.target.value)}>
            {ASSET_TYPES.map((t) => <option key={t} value={t}>{ASSET_TYPE_LABEL[t] ?? t}</option>)}
          </select>
        </div>
        <div className="field">
          <label>Valor atual estimado (R$)</label>
          <input type="number" step="0.01" min="0" value={currentValue} onChange={(e) => setCurrentValue(e.target.value)} />
          {fieldErrors.current_value && <span className="field-error">{fieldErrors.current_value}</span>}
        </div>
        <div className="field">
          <label>Data de referência</label>
          <input type="date" value={valuationDate} onChange={(e) => setValuationDate(e.target.value)} />
        </div>
        <div className="field" style={{ gridColumn: '1 / -1' }}>
          <label>Observações (opcional)</label>
          <textarea rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
        </div>
      </div>
      <div className="modal__actions">
        <button type="button" className="btn btn--secondary" onClick={onClose}>Cancelar</button>
        <button type="button" className="btn" disabled={saving} onClick={handleSubmit}>{saving ? 'Salvando...' : 'Criar'}</button>
      </div>
    </Modal>
  )
}

function ValuationFormModal({ item, onClose, onSaved }: { item: Asset; onClose: () => void; onSaved: () => void }) {
  const [value, setValue] = useState('')
  const [date, setDate] = useState(todayISO())
  const [notes, setNotes] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  async function handleSubmit() {
    setSaving(true)
    try {
      await assetsApi.addValuation(item.id, { value, valuation_date: date, notes: notes || null })
      onSaved()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Erro ao registrar valor.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={`Atualizar valor — ${item.name}`} onClose={onClose}>
      {error && <div className="alert alert--error">{error}</div>}
      <p className="stat-tile__hint" style={{ marginTop: 0 }}>O valor anterior fica guardado no histórico.</p>
      <div className="form-grid">
        <div className="field">
          <label htmlFor="val-value">Novo valor estimado (R$)</label>
          <input id="val-value" type="number" step="0.01" min="0" value={value} onChange={(e) => setValue(e.target.value)} />
        </div>
        <div className="field">
          <label htmlFor="val-date">Data de referência</label>
          <input id="val-date" type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </div>
        <div className="field" style={{ gridColumn: '1 / -1' }}>
          <label htmlFor="val-notes">Observações (opcional)</label>
          <input id="val-notes" value={notes} onChange={(e) => setNotes(e.target.value)} />
        </div>
      </div>
      <div className="modal__actions">
        <button type="button" className="btn btn--secondary" onClick={onClose}>Cancelar</button>
        <button type="button" className="btn" disabled={saving} onClick={handleSubmit}>{saving ? 'Salvando...' : 'Registrar'}</button>
      </div>
    </Modal>
  )
}

function HistoryModal({ item, onClose }: { item: Asset; onClose: () => void }) {
  const [history, setHistory] = useState<Valuation[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    assetsApi.listValuations(item.id)
      .then(setHistory)
      .catch((err) => setError(err instanceof ApiError ? err.message : 'Erro ao carregar histórico.'))
  }, [item.id])

  return (
    <Modal title={`Histórico — ${item.name}`} onClose={onClose}>
      {error && <div className="alert alert--error">{error}</div>}
      {history === null ? <LoadingState /> : history.length === 0 ? (
        <p>Nenhum registro de histórico.</p>
      ) : (
        <table>
          <thead><tr><th>Data</th><th>Valor</th><th>Observações</th></tr></thead>
          <tbody>
            {history.map((h) => (
              <tr key={h.id}>
                <td>{formatDateBR(h.valuation_date)}</td>
                <td>{formatBRL(h.value)}</td>
                <td>{h.notes ?? '—'}</td>
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
