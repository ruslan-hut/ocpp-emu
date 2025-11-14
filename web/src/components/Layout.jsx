import { Link, useLocation } from 'react-router-dom'
import './Layout.css'

function Layout({ children }) {
  const location = useLocation()

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
          </nav>
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
