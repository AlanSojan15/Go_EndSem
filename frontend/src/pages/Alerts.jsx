import { useEffect, useState } from 'react'
import { checkAlerts, createAlert, deleteAlert, getAlerts } from '../api'

// Inline SVG icons — no external library
const IconBell = () => (
  <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" style={{ verticalAlign: 'middle', marginRight: 5 }}>
    <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/>
    <path d="M13.73 21a2 2 0 0 1-3.46 0"/>
  </svg>
)

const IconTrash = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
    <polyline points="3 6 5 6 21 6"/>
    <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
    <path d="M10 11v6M14 11v6"/>
    <path d="M9 6V4h6v2"/>
  </svg>
)

function friendlyError(msg) {
  if (!msg) return 'Something went wrong.'
  const lower = msg.toLowerCase()
  if (lower.includes('mongo') || lower.includes('database') || lower.includes('server selection') || lower.includes('i/o timeout') || lower.includes('connect')) {
    return 'Could not connect to the database. Please check your MongoDB connection or try again later.'
  }
  if (lower.includes('rate limit') || lower.includes('429')) {
    return 'API rate limit reached. Please wait a few seconds and try again.'
  }
  return msg
}

export default function Alerts({ token }) {
  const [alerts, setAlerts] = useState([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [form, setForm] = useState({ coin_id: '', alert_type: 'buy', threshold: '' })
  const [creating, setCreating] = useState(false)
  const [checking, setChecking] = useState(false)
  const [deletingId, setDeletingId] = useState(null)

  useEffect(() => {
    if (!token) return
    loadAlerts()
  }, [token])

  async function loadAlerts() {
    setError('')
    try {
      const data = await getAlerts(token)
      setAlerts(data || [])
    } catch (err) {
      setError(friendlyError(err.message))
    }
  }

  async function handleCreate(e) {
    e.preventDefault()
    setError('')
    setInfo('')
    setCreating(true)
    try {
      await createAlert(token, { ...form, threshold: Number(form.threshold) })
      setInfo('✅ Alert created successfully!')
      setForm({ coin_id: '', alert_type: 'buy', threshold: '' })
      loadAlerts()
    } catch (err) {
      setError(friendlyError(err.message))
    } finally {
      setCreating(false)
    }
  }

  async function handleCheck() {
    setError('')
    setInfo('')
    setChecking(true)
    try {
      await checkAlerts(token)
      setInfo('✅ Alerts checked — any triggered alerts have been emailed to you.')
      loadAlerts()
    } catch (err) {
      setError(friendlyError(err.message))
    } finally {
      setChecking(false)
    }
  }

  async function handleDelete(alertId) {
    setError('')
    setDeletingId(alertId)
    try {
      await deleteAlert(token, alertId)
      setInfo('🗑️ Alert deleted.')
      setAlerts((prev) => prev.filter((a) => a.id !== alertId))
    } catch (err) {
      setError(friendlyError(err.message))
    } finally {
      setDeletingId(null)
    }
  }

  if (!token) return <p>Please log in to manage alerts.</p>

  return (
    <div>
      <h2>Price Alerts</h2>
      {error && <div className="alert error">{error}</div>}
      {info && <div className="alert success">{info}</div>}

      <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
        <div style={{ padding: '16px 20px', borderBottom: '1px solid var(--border)', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <strong style={{ fontSize: '0.95rem' }}>Active Alerts</strong>
          <button
            onClick={handleCheck}
            disabled={checking}
            className="button-inline"
            style={{ minWidth: 160 }}
          >
            {checking ? 'Checking…' : <><IconBell />Check Alerts Now</>}
          </button>
        </div>

        {alerts.length === 0 ? (
          <p className="empty-state">No active alerts — create one below.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Coin</th>
                <th>Type</th>
                <th>Threshold</th>
                <th>Created</th>
                <th style={{ width: 60 }}></th>
              </tr>
            </thead>
            <tbody>
              {alerts.map((a) => (
                <tr key={a.id}>
                  <td>
                    <strong>{a.coin_name}</strong>
                    <span style={{ color: 'var(--muted)', fontSize: '0.8em', marginLeft: 6 }}>{a.coin_id}</span>
                  </td>
                  <td>
                    <span className={`badge ${a.alert_type === 'buy' ? 'badge-green' : 'badge-red'}`}>
                      {a.alert_type === 'buy' ? '↓ BUY' : '↑ SELL'}
                    </span>
                  </td>
                  <td><strong>${Number(a.threshold_price ?? a.threshold).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</strong></td>
                  <td style={{ color: 'var(--muted)', fontSize: '0.82rem' }}>
                    {a.created_at ? new Date(a.created_at).toLocaleDateString() : '—'}
                  </td>
                  <td>
                    <button
                      onClick={() => handleDelete(a.id)}
                      disabled={deletingId === a.id}
                      className="btn-danger-sm"
                      title="Delete alert"
                    >
                      {deletingId === a.id ? '…' : <IconTrash />}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="card">
        <h3>Create Alert</h3>
        <form onSubmit={handleCreate}>
          <label>Coin ID <span style={{ color: 'var(--muted)', fontWeight: 400 }}>(must be in your portfolio, e.g. bitcoin)</span></label>
          <input
            value={form.coin_id}
            placeholder="bitcoin"
            onChange={(e) => setForm((s) => ({ ...s, coin_id: e.target.value }))}
            required
          />

          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
            <div>
              <label>Alert Type</label>
              <select
                value={form.alert_type}
                onChange={(e) => setForm((s) => ({ ...s, alert_type: e.target.value }))}
              >
                <option value="buy">▼ Buy (price drops to)</option>
                <option value="sell">▲ Sell (price rises to)</option>
              </select>
            </div>
            <div>
              <label>Threshold Price ($)</label>
              <input
                type="number"
                step="any"
                min="0"
                value={form.threshold}
                placeholder="50000"
                onChange={(e) => setForm((s) => ({ ...s, threshold: e.target.value }))}
                required
              />
            </div>
          </div>

          <button type="submit" disabled={creating} style={{ marginTop: 4 }}>
            {creating ? 'Creating…' : '+ Create Alert'}
          </button>
        </form>
      </div>
    </div>
  )
}
