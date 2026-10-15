import { useState } from 'react'
import { useAuth } from '../context/AuthContext'
import { profileApi } from '../api/endpoints'
import { ApiError } from '../api/client'

export default function ProfilePage() {
  const { user, refreshProfile } = useAuth()
  const [name, setName] = useState(user?.name ?? '')
  const [email, setEmail] = useState(user?.email ?? '')
  const [profileError, setProfileError] = useState<string | null>(null)
  const [profileSuccess, setProfileSuccess] = useState(false)
  const [profileFieldErrors, setProfileFieldErrors] = useState<Record<string, string>>({})
  const [savingProfile, setSavingProfile] = useState(false)

  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [passwordSuccess, setPasswordSuccess] = useState(false)
  const [savingPassword, setSavingPassword] = useState(false)

  async function handleProfileSubmit() {
    setSavingProfile(true)
    setProfileError(null)
    setProfileFieldErrors({})
    setProfileSuccess(false)
    try {
      await profileApi.update({ name, email })
      await refreshProfile()
      setProfileSuccess(true)
    } catch (err) {
      if (err instanceof ApiError && err.fields) setProfileFieldErrors(err.fields)
      setProfileError(err instanceof ApiError ? err.message : 'Erro ao atualizar perfil.')
    } finally {
      setSavingProfile(false)
    }
  }

  async function handlePasswordSubmit() {
    setSavingPassword(true)
    setPasswordError(null)
    setPasswordSuccess(false)
    try {
      await profileApi.changePassword({ current_password: currentPassword, new_password: newPassword })
      setCurrentPassword('')
      setNewPassword('')
      setPasswordSuccess(true)
    } catch (err) {
      setPasswordError(err instanceof ApiError ? err.message : 'Erro ao alterar senha.')
    } finally {
      setSavingPassword(false)
    }
  }

  return (
    <div>
      <div className="page-header">
        <h1>Perfil e configurações</h1>
      </div>

      <div className="grid grid--two">
        <div className="card">
          <h3>Dados pessoais</h3>
          {profileError && <div className="alert alert--error">{profileError}</div>}
          {profileSuccess && <div className="alert alert--success">Perfil atualizado com sucesso.</div>}
          <div className="field" style={{ marginBottom: '0.9rem' }}>
            <label>Nome</label>
            <input value={name} onChange={(e) => setName(e.target.value)} />
            {profileFieldErrors.name && <span className="field-error">{profileFieldErrors.name}</span>}
          </div>
          <div className="field" style={{ marginBottom: '1rem' }}>
            <label>E-mail</label>
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            {profileFieldErrors.email && <span className="field-error">{profileFieldErrors.email}</span>}
          </div>
          <button type="button" className="btn" disabled={savingProfile} onClick={handleProfileSubmit}>
            {savingProfile ? 'Salvando...' : 'Salvar alterações'}
          </button>
        </div>

        <div className="card">
          <h3>Alterar senha</h3>
          {passwordError && <div className="alert alert--error">{passwordError}</div>}
          {passwordSuccess && <div className="alert alert--success">Senha alterada com sucesso.</div>}
          <div className="field" style={{ marginBottom: '0.9rem' }}>
            <label>Senha atual</label>
            <input type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} autoComplete="current-password" />
          </div>
          <div className="field" style={{ marginBottom: '1rem' }}>
            <label>Nova senha</label>
            <input type="password" minLength={8} value={newPassword} onChange={(e) => setNewPassword(e.target.value)} autoComplete="new-password" />
          </div>
          <button type="button" className="btn" disabled={savingPassword} onClick={handlePasswordSubmit}>
            {savingPassword ? 'Salvando...' : 'Alterar senha'}
          </button>
        </div>
      </div>

      <div className="card" style={{ marginTop: '1rem' }}>
        <h3>Recuperação de senha e verificação de e-mail</h3>
        <p style={{ fontSize: '0.88rem', color: 'var(--color-text-muted)' }}>
          Estas funcionalidades dependem de um serviço de envio de e-mail, que não está configurado neste
          ambiente. Por isso, não são exibidas como disponíveis — altere sua senha diretamente aqui, estando
          autenticado.
        </p>
      </div>
    </div>
  )
}
