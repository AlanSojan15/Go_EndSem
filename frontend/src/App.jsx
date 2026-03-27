import { useMemo, useState } from 'react'
import { BrowserRouter, Link, Navigate, Route, Routes } from 'react-router-dom'
import Login from './pages/Login'
import Signup from './pages/Signup'
import Portfolio from './pages/Portfolio'
import Alerts from './pages/Alerts'
import './App.css'

// SVG icon components — inline, no external library needed
const IconCoin = () => (
  <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="10"/>
    <path d="M12 6v2m0 8v2M9 9.5C9 8.1 10.3 7 12 7s3 1.1 3 2.5-1.3 2.5-3 2.5-3 1.1-3 2.5S10.3 17 12 17s3-1.1 3-2.5"/>
  </svg>
)

const IconLogout = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
    <polyline points="16 17 21 12 16 7"/>
    <line x1="21" y1="12" x2="9" y2="12"/>
  </svg>
)

function App() {
  const [token, setToken] = useState(localStorage.getItem('token') || '')

  const isAuthenticated = useMemo(() => Boolean(token), [token])

  function handleLogin(newToken) {
    setToken(newToken)
    localStorage.setItem('token', newToken)
  }

  function handleLogout() {
    setToken('')
    localStorage.removeItem('token')
  }

  return (
    <BrowserRouter>
      <header className="app-header">
        {/* Brand name changed to CoinVault for unique identity */}
        <div className="brand">
          <IconCoin />
          CoinVault
        </div>
        <nav>
          {isAuthenticated ? (
            <>
              <Link to="/portfolio">Portfolio</Link>
              <Link to="/alerts">Alerts</Link>
              <button className="link-button" onClick={handleLogout}>
                <IconLogout /> Logout
              </button>
            </>
          ) : (
            <>
              <Link to="/login">Login</Link>
              <Link to="/signup">Sign up</Link>
            </>
          )}
        </nav>
      </header>

      <main className="app-main">
        <Routes>
          <Route path="/" element={<Navigate to={isAuthenticated ? '/portfolio' : '/login'} replace />} />
          <Route
            path="/login"
            element={
              isAuthenticated ? (
                <Navigate to="/portfolio" replace />
              ) : (
                <Login onLogin={handleLogin} />
              )
            }
          />
          <Route
            path="/signup"
            element={isAuthenticated ? <Navigate to="/portfolio" replace /> : <Signup />}
          />
          <Route
            path="/portfolio"
            element={isAuthenticated ? <Portfolio token={token} /> : <Navigate to="/login" replace />}
          />
          <Route
            path="/alerts"
            element={isAuthenticated ? <Alerts token={token} /> : <Navigate to="/login" replace />}
          />
        </Routes>
      </main>

      <footer className="app-footer">
        <span>CoinVault · Developed by <strong>Alan Sojan</strong> · March 2026</span>
      </footer>
    </BrowserRouter>
  )
}

export default App
