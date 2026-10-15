// Utilitários de formatação pt-BR. Valores monetários são recebidos como
// string decimal da API e convertidos para number apenas no momento da
// exibição (nunca para cálculos no cliente).

const currencyFormatter = new Intl.NumberFormat('pt-BR', { style: 'currency', currency: 'BRL' })
const percentFormatter = new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })

export function formatBRL(value: string | number | null | undefined): string {
  if (value === null || value === undefined) return '—'
  const n = typeof value === 'string' ? Number(value) : value
  if (Number.isNaN(n)) return '—'
  return currencyFormatter.format(n)
}

export function formatPercent(value: string | number | null | undefined): string {
  if (value === null || value === undefined) return '—'
  const n = typeof value === 'string' ? Number(value) : value
  if (Number.isNaN(n)) return '—'
  return `${percentFormatter.format(n)}%`
}

// Converte uma data no formato AAAA-MM-DD (ou ISO com horário) para o
// formato brasileiro DD/MM/AAAA, sem depender do fuso horário local do
// navegador (evita o clássico erro de "um dia a menos").
export function formatDateBR(isoDate: string | null | undefined): string {
  if (!isoDate) return '—'
  const datePart = isoDate.slice(0, 10)
  const [year, month, day] = datePart.split('-')
  if (!year || !month || !day) return isoDate
  return `${day}/${month}/${year}`
}

export function formatDateTimeBR(isoDateTime: string | null | undefined): string {
  if (!isoDateTime) return '—'
  const d = new Date(isoDateTime)
  if (Number.isNaN(d.getTime())) return isoDateTime
  return d.toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
}

export function todayISO(): string {
  const d = new Date()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${d.getFullYear()}-${mm}-${dd}`
}

export const MONTH_NAMES_PT = [
  'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
  'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro',
]

export function monthLabel(month: number): string {
  return MONTH_NAMES_PT[month - 1] ?? String(month)
}

export const CATEGORY_KIND_LABEL: Record<string, string> = {
  income: 'Receita',
  expense: 'Despesa',
}

export const TRANSACTION_TYPE_LABEL: Record<string, string> = {
  income: 'Receita',
  expense: 'Despesa',
  transfer_in: 'Transferência (entrada)',
  transfer_out: 'Transferência (saída)',
}

export const TRANSACTION_STATUS_LABEL: Record<string, string> = {
  planned: 'Planejada',
  pending: 'Pendente',
  paid: 'Paga/Recebida',
  cancelled: 'Cancelada',
}

export const ASSET_TYPE_LABEL: Record<string, string> = {
  imovel: 'Imóveis',
  veiculo: 'Veículos',
  participacao_empresarial: 'Participações empresariais',
  dinheiro_em_especie: 'Dinheiro em espécie',
  outro: 'Outros bens',
}

export const INVESTMENT_TYPE_LABEL: Record<string, string> = {
  renda_fixa: 'Renda fixa',
  tesouro_direto: 'Tesouro Direto',
  cdb: 'CDB',
  lci: 'LCI',
  lca: 'LCA',
  fundo: 'Fundos',
  acao: 'Ações',
  etf: 'ETFs',
  fii: 'Fundos imobiliários',
  previdencia: 'Previdência',
  cripto: 'Criptoativos',
  outro: 'Outros investimentos',
}

export const PERIODICITY_LABEL: Record<string, string> = {
  monthly: 'Mensal',
  yearly: 'Anual',
  weekly: 'Semanal',
}
