import { Navigate, Outlet } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { LoadingState } from './States'

export function ProtectedRoute() {
  const { user, loading } = useAuth()

  if (loading) return <LoadingState label="Verificando sessão..." />
  if (!user) return <Navigate to="/login" replace />

  return <Outlet />
}
