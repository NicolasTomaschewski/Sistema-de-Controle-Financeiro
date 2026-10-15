// Até 6 fatias, cada uma com uma cor fixa da paleta categórica (validada para
// daltonismo sobre a superfície escura, inclusive o par que fecha o círculo).
export const MAX_SLICES = 6

export type PieSlice = { key: string; label: string; value: number; share: number; valueText: string; shareText: string }

// Recebe as fatias ordenadas da maior para a menor (como vêm da API).
export function groupSmallSlices(
  slices: PieSlice[],
  formatValue: (value: number) => string,
  formatShare: (share: number) => string,
): PieSlice[] {
  if (slices.length <= MAX_SLICES) return slices
  const head = slices.slice(0, MAX_SLICES - 1)
  const rest = slices.slice(MAX_SLICES - 1)
  const value = rest.reduce((sum, s) => sum + s.value, 0)
  const share = rest.reduce((sum, s) => sum + s.share, 0)
  return [...head, { key: '__outros', label: `Outros (${rest.length})`, value, share, valueText: formatValue(value), shareText: formatShare(share) }]
}
