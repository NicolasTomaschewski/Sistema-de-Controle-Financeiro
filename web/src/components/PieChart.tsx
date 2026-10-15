import { useState } from 'react'
import { MAX_SLICES, type PieSlice } from '../utils/pie'

// Gráfico de pizza (rosca) em SVG puro. Recebe fatias já agregadas e com a
// participação calculada pela API; o número convertido aqui só posiciona os
// arcos. Até 6 fatias, cada uma com uma cor fixa da paleta categórica
// (validada para daltonismo sobre a superfície escura, inclusive o par que
// fecha o círculo). Acima disso, as menores são agrupadas em "Outros".

const SIZE = 220
const OUTER = 104
const INNER = 66

function arcPath(start: number, end: number): string {
  const c = SIZE / 2
  const pt = (r: number, a: number) => `${c + r * Math.sin(a)} ${c - r * Math.cos(a)}`
  const large = end - start > Math.PI ? 1 : 0
  return [
    `M ${pt(OUTER, start)}`,
    `A ${OUTER} ${OUTER} 0 ${large} 1 ${pt(OUTER, end)}`,
    `L ${pt(INNER, end)}`,
    `A ${INNER} ${INNER} 0 ${large} 0 ${pt(INNER, start)}`,
    'Z',
  ].join(' ')
}

export function PieChart({ slices, totalText, totalLabel, ariaLabel }: {
  slices: PieSlice[]
  totalText: string
  totalLabel: string
  ariaLabel: string
}) {
  const [active, setActive] = useState<number | null>(null)
  const total = slices.reduce((sum, s) => sum + s.value, 0)
  const visible = slices.slice(0, MAX_SLICES)

  const arcs = visible.reduce<{ start: number; end: number }[]>((acc, s) => {
    const start = acc.length > 0 ? acc[acc.length - 1].end : 0
    return [...acc, { start, end: start + (total > 0 ? (s.value / total) * Math.PI * 2 : 0) }]
  }, [])

  const focus = active !== null ? visible[active] : null

  return (
    <div className="pie">
      <svg viewBox={`0 0 ${SIZE} ${SIZE}`} className="pie__svg" role="img" aria-label={ariaLabel}>
        {visible.map((s, i) => {
          const { start, end } = arcs[i]
          const dim = active !== null && active !== i
          const common = {
            className: `pie__slice pie__slice--${i + 1}${dim ? ' pie__slice--dim' : ''}`,
            onMouseEnter: () => setActive(i),
            onMouseLeave: () => setActive(null),
          }
          // Um único arco de 360° não é desenhável com "A"; usa-se um anel completo.
          return end - start >= Math.PI * 2 - 1e-6 ? (
            <circle key={s.key} {...common} cx={SIZE / 2} cy={SIZE / 2} r={(OUTER + INNER) / 2} style={{ fill: 'none', stroke: `var(--cat-${i + 1})` }} />
          ) : (
            <path key={s.key} {...common} d={arcPath(start, end)} />
          )
        })}
        <text x={SIZE / 2} y={SIZE / 2 - 6} textAnchor="middle" className="pie__center-label">
          {focus ? focus.label : totalLabel}
        </text>
        <text x={SIZE / 2} y={SIZE / 2 + 16} textAnchor="middle" className="pie__center-value">
          {focus ? focus.shareText : totalText}
        </text>
      </svg>

      <ul className="pie__legend">
        {visible.map((s, i) => (
          <li
            key={s.key}
            className={`pie__legend-item${active === i ? ' pie__legend-item--active' : ''}`}
            tabIndex={0}
            onMouseEnter={() => setActive(i)}
            onMouseLeave={() => setActive(null)}
            onFocus={() => setActive(i)}
            onBlur={() => setActive(null)}
          >
            <span className={`chart-legend__swatch pie__swatch--${i + 1}`} aria-hidden="true" />
            <span className="pie__legend-label">{s.label}</span>
            <span className="pie__legend-value">{s.valueText}</span>
            <span className="pie__legend-share">{s.shareText}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
