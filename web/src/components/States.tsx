export function LoadingState({ label = 'Carregando...' }: { label?: string }) {
  return <div className="empty-state">{label}</div>
}

export function ErrorState({ message }: { message: string }) {
  return <div className="alert alert--error">{message}</div>
}

export function EmptyState({ title, hint }: { title: string; hint?: string }) {
  return (
    <div className="empty-state">
      <div style={{ fontWeight: 600, marginBottom: hint ? '0.3rem' : 0 }}>{title}</div>
      {hint && <div style={{ fontSize: '0.85rem' }}>{hint}</div>}
    </div>
  )
}
