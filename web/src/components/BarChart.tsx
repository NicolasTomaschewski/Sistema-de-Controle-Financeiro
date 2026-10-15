// Gráfico de barras simples em SVG puro, sem dependência externa de
// biblioteca de gráficos — suficiente para as poucas séries exibidas aqui.

type Series = { label: string; value: number; color: string }
type Group = { label: string; series: Series[] }

export function GroupedBarChart({ groups, height = 220 }: { groups: Group[]; height?: number }) {
  const allValues = groups.flatMap((g) => g.series.map((s) => s.value))
  const max = Math.max(1, ...allValues.map(Math.abs))
  const barWidth = 18
  const groupGap = 28
  const seriesGap = 4
  const width = groups.length * (groupGap + groups[0]?.series.length * (barWidth + seriesGap) || 0) + 20

  return (
    <div style={{ overflowX: 'auto' }}>
      <svg width={Math.max(width, 320)} height={height + 40} role="img" aria-label="Gráfico de evolução mensal">
        {groups.map((group, gi) => {
          const groupX = 10 + gi * (groupGap + group.series.length * (barWidth + seriesGap))
          return (
            <g key={group.label}>
              {group.series.map((s, si) => {
                const barHeight = (Math.abs(s.value) / max) * height
                const x = groupX + si * (barWidth + seriesGap)
                const y = height - barHeight
                return (
                  <rect key={s.label} x={x} y={y} width={barWidth} height={barHeight} style={{ fill: s.color }} rx={2}>
                    <title>{`${s.label}: ${s.value.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })}`}</title>
                  </rect>
                )
              })}
              <text x={groupX + (group.series.length * (barWidth + seriesGap)) / 2} y={height + 18} fontSize="11" textAnchor="middle" style={{ fill: 'var(--color-text-muted)' }}>
                {group.label}
              </text>
            </g>
          )
        })}
      </svg>
    </div>
  )
}
