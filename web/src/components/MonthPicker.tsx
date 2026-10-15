import { monthLabel } from '../utils/format'

export function MonthPicker({
  year,
  month,
  onChange,
}: {
  year: number
  month: number
  onChange: (year: number, month: number) => void
}) {
  function shift(delta: number) {
    let y = year
    let m = month + delta
    if (m < 1) { m = 12; y -= 1 }
    if (m > 12) { m = 1; y += 1 }
    onChange(y, m)
  }

  return (
    <div className="toolbar" aria-label="Seleção de mês e ano">
      <button type="button" className="btn btn--secondary btn--sm" onClick={() => shift(-1)} aria-label="Mês anterior">
        ←
      </button>
      <strong style={{ minWidth: '140px', textAlign: 'center' }}>{monthLabel(month)} de {year}</strong>
      <button type="button" className="btn btn--secondary btn--sm" onClick={() => shift(1)} aria-label="Próximo mês">
        →
      </button>
    </div>
  )
}
