const API_BASE = import.meta.env.VITE_API_BASE ?? ''

async function request(path, { method = 'GET', token, body } = {}) {
  const headers = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`

  const res = await fetch(`${API_BASE}/api${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })

  const json = await res.json().catch(() => ({}))
  if (!res.ok || (json && json.success === false)) {
    throw new Error(json.error || 'Request failed')
  }
  return json.data
}

async function requestRaw(path, { method = 'GET', token, body } = {}) {
  const headers = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`

  const res = await fetch(`${API_BASE}/api${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })

  if (!res.ok) {
    const json = await res.json().catch(() => ({}))
    throw new Error(json.error || 'Request failed')
  }
  return res
}

export async function signup(email, password) {
  return request('/signup', { method: 'POST', body: { email, password } })
}

export async function login(email, password) {
  return request('/login', { method: 'POST', body: { email, password } })
}

export async function getPortfolio(token) {
  return request('/portfolio', { method: 'GET', token })
}

export async function addHolding(token, holding) {
  return request('/portfolio/holdings', { method: 'POST', token, body: holding })
}

export async function exportPortfolio(token) {
  const res = await requestRaw('/portfolio/export', { method: 'GET', token })
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'portfolio.json'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

export async function importPortfolio(token, holdings) {
  return request('/portfolio/import', { method: 'POST', token, body: { holdings } })
}

export async function getAlerts(token) {
  return request('/alerts', { method: 'GET', token })
}

export async function createAlert(token, alert) {
  return request('/alerts', { method: 'POST', token, body: alert })
}

export async function deleteAlert(token, id) {
  return request('/alerts/delete', { method: 'POST', token, body: { id } })
}

export async function checkAlerts(token) {
  return request('/alerts/check', { method: 'POST', token })
}

// ──────────────────────────────────────────────────────────────────
// ENHANCED FEATURE 2: Fear & Greed Index
// Fetches the current Crypto Fear & Greed Index (0-100) from
// alternative.me — a free API, no key required.
// Used in Portfolio.jsx to display market sentiment to the user.
// ──────────────────────────────────────────────────────────────────
export async function getFearGreedIndex() {
  const res = await fetch('https://api.alternative.me/fng/?limit=1')
  if (!res.ok) throw new Error('Failed to fetch Fear & Greed Index')
  const json = await res.json()
  return json.data[0] // { value, value_classification, timestamp }
}

// ──────────────────────────────────────────────────────────────────
// ENHANCED FEATURE 3: Multi-Currency Support
// Fetches live USD exchange rates from exchangerate-api.com (free).
// Used in Portfolio.jsx to convert all prices to USD / INR / EUR.
// ──────────────────────────────────────────────────────────────────
export async function getExchangeRates() {
  const res = await fetch('https://api.exchangerate-api.com/v4/latest/USD')
  if (!res.ok) throw new Error('Failed to fetch exchange rates')
  const json = await res.json()
  return json.rates // { USD: 1, INR: 83.5, EUR: 0.92, ... }
}
