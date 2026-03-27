import { useMemo, useState } from 'react'
import { BrowserRouter, Link, Navigate, Route, Routes } from 'react-router-dom'
import Login from './pages/Login'
import Signup from './pages/Signup'
import Portfolio from './pages/Portfolio'
import Alerts from './pages/Alerts'
import './App.css'

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
        <div className="brand">Crypto Portfolio Tracker</div>
        <nav>
          {isAuthenticated ? (
            <>
              <Link to="/portfolio">Portfolio</Link>
              <Link to="/alerts">Alerts</Link>
              <button className="link-button" onClick={handleLogout}>
                Logout
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
        <span>Developed by <strong>Alok</strong> and <strong>Alan</strong> · March 2026</span>
      </footer>
    </BrowserRouter>
  )
}

export default App
