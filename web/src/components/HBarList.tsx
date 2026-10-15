// Lista de barras horizontais de uma única série (ex.: receitas por fonte,
// custo por categoria). Os textos exibidos vêm formatados da API; o número
// serve apenas para o comprimento relativo da barra.

export type HBarItem = { key: string; label: string; value: number; valueText: string; detail?: string }

export function HBarList({ items, tone, emptyText, ariaLabel }: {
  items: HBarItem[]
  tone: 'increase' | 'decrease'
  emptyText: string
  ariaLabel: string
}) {
  const max = Math.max(0, ...items.map((i) => i.value))
  if (items.length === 0 || max <= 0) {
    return <p className="hbar__empty">{emptyText}</p>
  }

  return (
    <ul className="hbar" aria-label={ariaLabel}>
      {items.map((item) => (
        <li key={item.key} className="hbar__row" title={`${item.label}: ${item.valueText}${item.detail ? ` (${item.detail})` : ''}`}>
          <div className="hbar__head">
            <span className="hbar__label">{item.label}</span>
            <span className="hbar__value">
              {item.valueText}
              {item.detail && <span className="hbar__detail"> · {item.detail}</span>}
            </span>
          </div>
          <div className="hbar__track">
            <div className={`hbar__fill hbar__fill--${tone}`} style={{ width: `max(${(item.value / max) * 100}%, 2px)` }} />
          </div>
        </li>
      ))}
    </ul>
  )
}
