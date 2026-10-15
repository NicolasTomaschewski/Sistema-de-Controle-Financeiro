import type { Category, Transaction } from '../api/types'
import { formatBRL, formatDateBR, TRANSACTION_STATUS_LABEL } from '../utils/format'
import { EmptyState } from './States'

export function TransactionsTable({
  transactions,
  categories,
  onEdit,
  onDelete,
}: {
  transactions: Transaction[]
  categories: Category[]
  onEdit: (t: Transaction) => void
  onDelete: (t: Transaction) => void
}) {
  if (transactions.length === 0) {
    return <EmptyState title="Nenhum lançamento encontrado" hint="Ajuste os filtros ou cadastre um novo lançamento." />
  }

  const categoryName = (id: string | null) => (id ? categories.find((c) => c.id === id)?.name ?? '—' : '—')

  return (
    <div className="table-wrap card">
      <table>
        <thead>
          <tr>
            <th>Data</th><th>Descrição</th><th>Categoria</th><th>Status</th><th>Valor</th><th></th>
          </tr>
        </thead>
        <tbody>
          {transactions.map((t) => (
            <tr key={t.id}>
              <td>{formatDateBR(t.occurred_on)}</td>
              <td>{t.description}</td>
              <td>{categoryName(t.category_id)}</td>
              <td><span className={`badge badge--${t.status}`}>{TRANSACTION_STATUS_LABEL[t.status]}</span></td>
              <td>{formatBRL(t.amount)}</td>
              <td>
                <button type="button" className="btn btn--secondary btn--sm" onClick={() => onEdit(t)}>Editar</button>{' '}
                <button type="button" className="btn btn--danger btn--sm" onClick={() => onDelete(t)}>Excluir</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
