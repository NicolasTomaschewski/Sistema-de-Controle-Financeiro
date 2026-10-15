import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { ApiError } from '../api/client'

export default function RegisterPage() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [submitting, setSubmitting] = useState(false)
  const [success, setSuccess] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setFieldErrors({})
    setSubmitting(true)
    try {
      await register(name, email, password)
      setSuccess(true)
      setTimeout(() => navigate('/login', { replace: true }), 1200)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
        if (err.fields) setFieldErrors(err.fields)
      } else {
        setError('Não foi possível concluir o cadastro.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <h2>Criar conta</h2>
        {error && <div className="alert alert--error">{error}</div>}
        {success && <div className="alert alert--success">Cadastro realizado! Redirecionando para o login...</div>}
        <form onSubmit={handleSubmit}>
          <div className="field" style={{ marginBottom: '0.9rem' }}>
            <label htmlFor="name">Nome</label>
            <input id="name" required value={name} onChange={(e) => setName(e.target.value)} autoComplete="name" />
            {fieldErrors.name && <span className="field-error">{fieldErrors.name}</span>}
          </div>
          <div className="field" style={{ marginBottom: '0.9rem' }}>
            <label htmlFor="email">E-mail</label>
            <input id="email" type="email" required value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" />
            {fieldErrors.email && <span className="field-error">{fieldErrors.email}</span>}
          </div>
          <div className="field" style={{ marginBottom: '1.2rem' }}>
            <label htmlFor="password">Senha</label>
            <input id="password" type="password" required minLength={8} value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" />
            {fieldErrors.password && <span className="field-error">{fieldErrors.password}</span>}
            <span style={{ fontSize: '0.78rem', color: 'var(--color-text-muted)' }}>Mínimo de 8 caracteres.</span>
          </div>
          <button type="submit" className="btn" style={{ width: '100%' }} disabled={submitting}>
            {submitting ? 'Enviando...' : 'Cadastrar'}
          </button>
        </form>
        <p style={{ marginTop: '1rem', fontSize: '0.88rem' }}>
          Já tem uma conta? <Link to="/login">Entrar</Link>
        </p>
      </div>
    </div>
  )
}
