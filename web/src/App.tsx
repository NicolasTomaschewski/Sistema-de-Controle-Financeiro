import { Navigate, Route, Routes } from 'react-router-dom'
import { Layout } from './components/Layout'
import { ProtectedRoute } from './components/ProtectedRoute'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import DashboardPage from './pages/DashboardPage'
import ExpensesPage from './pages/ExpensesPage'
import IncomePage from './pages/IncomePage'
import NetWorthPage from './pages/NetWorthPage'
import ReportsPage from './pages/ReportsPage'
import ProfilePage from './pages/ProfilePage'

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/registro" element={<RegisterPage />} />

      <Route element={<ProtectedRoute />}>
        <Route element={<Layout />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/receitas" element={<IncomePage />} />
          <Route path="/despesas" element={<ExpensesPage />} />
          <Route path="/gastos-mensais" element={<Navigate to="/despesas" replace />} />
          <Route path="/contas" element={<Navigate to="/" replace />} />
          <Route path="/categorias" element={<Navigate to="/" replace />} />
          <Route path="/patrimonio" element={<NetWorthPage />} />
          <Route path="/investimentos" element={<Navigate to="/patrimonio" replace />} />
          <Route path="/relatorios" element={<ReportsPage />} />
          <Route path="/perfil" element={<ProfilePage />} />
        </Route>
      </Route>
    </Routes>
  )
}
