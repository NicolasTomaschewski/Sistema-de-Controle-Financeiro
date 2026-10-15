import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { authApi, profileApi } from '../api/endpoints'
import { ApiError } from '../api/client'
import type { Profile } from '../api/types'

type AuthState = {
  user: Profile | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (name: string, email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  refreshProfile: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<Profile | null>(null)
  const [loading, setLoading] = useState(true)

  const refreshProfile = useCallback(async () => {
    try {
      const profile = await profileApi.get()
      setUser(profile)
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setUser(null)
      } else {
        throw err
      }
    }
  }, [])

  useEffect(() => {
    refreshProfile().finally(() => setLoading(false))
  }, [refreshProfile])

  const login = useCallback(async (email: string, password: string) => {
    await authApi.login({ email, password })
    await refreshProfile()
  }, [refreshProfile])

  const register = useCallback(async (name: string, email: string, password: string) => {
    await authApi.register({ name, email, password })
  }, [])

  const logout = useCallback(async () => {
    await authApi.logout()
    setUser(null)
  }, [])

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout, refreshProfile }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth deve ser usado dentro de um AuthProvider')
  return ctx
}
