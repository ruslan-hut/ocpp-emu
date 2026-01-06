import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import ThemeToggle from './ui/ThemeToggle'
import './Layout.css'

function Layout({ children }) {
  const location = useLocation()
  const { user, logout, isAdmin } = useAuth()

  const isActive = (path) => {
    return location.pathname === path ? 'active' : ''
  }

  return (
    <div className="app">
      <header className="header">
        <div className="header-content">
          <div className="logo">
            <h1>OCPP Emulator</h1>
            <span className="version">v0.1.0</span>
          </div>
          <nav className="nav">
            <Link to="/" className={`nav-link ${isActive('/')}`}>
              Dashboard
            </Link>
            <Link to="/stations" className={`nav-link ${isActive('/stations')}`}>
              Stations
            </Link>
            <Link to="/messages" className={`nav-link ${isActive('/messages')}`}>
              Messages
            </Link>
            <Link to="/message-crafter" className={`nav-link ${isActive('/message-crafter')}`}>
              Message Crafter
            </Link>
            <Link to="/scenarios" className={`nav-link ${isActive('/scenarios')}`}>
              Scenarios
            </Link>
          </nav>
          <div className="header-actions">
            {user && (
              <div className="user-info">
                <span className="username">{user.username}</span>
                <span className={`role-badge ${isAdmin ? 'role-admin' : 'role-viewer'}`}>
                  {user.role}
                </span>
                <button onClick={logout} className="logout-btn">
                  Logout
                </button>
              </div>
            )}
            <ThemeToggle />
          </div>
        </div>
      </header>
      <main className="main-content">
        {children}
      </main>
      <footer className="footer">
        <p>OCPP Charging Station Emulator</p>
      </footer>
    </div>
  )
}

export default Layout
