import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

const NAV_ITEMS = [
  { to: '/', label: 'Dashboard' },
  { to: '/receitas', label: 'Receitas' },
  { to: '/despesas', label: 'Despesas' },
  { to: '/patrimonio', label: 'Patrimônio' },
  { to: '/relatorios', label: 'Relatórios' },
  { to: '/perfil', label: 'Perfil' },
]

export function Layout() {
  const { user, logout } = useAuth()

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="sidebar__brand">Controle Financeiro</div>
        <nav className="sidebar__nav">
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) => 'sidebar__link' + (isActive ? ' sidebar__link--active' : '')}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar__footer">
          <div>{user?.name}</div>
          <button
            type="button"
            className="btn btn--secondary btn--sm"
            style={{ marginTop: '0.5rem', width: '100%' }}
            onClick={() => void logout()}
          >
            Sair
          </button>
        </div>
      </aside>
      <main className="main-content">
        <Outlet />
      </main>
    </div>
  )
}
