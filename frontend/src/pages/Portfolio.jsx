import { useEffect, useMemo, useRef, useState } from 'react'
import { addHolding, exportPortfolio, importPortfolio, getPortfolio, getFearGreedIndex, getExchangeRates } from '../api'

// ──────────────────────────────────────────────────────────────────
// ENHANCED FEATURE 1: Portfolio Allocation Pie Chart
// Imported from recharts library (npm install recharts)
// Renders a pie chart showing each coin's % share of total portfolio
// ──────────────────────────────────────────────────────────────────
import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from 'recharts'

// PIE_COLORS palette for pie chart slices
const PIE_COLORS = ['#6366f1', '#f59e0b', '#10b981', '#ef4444', '#3b82f6', '#ec4899', '#14b8a6', '#f97316']

// SVG chart icon used in the pie chart section title
const IconChart = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ verticalAlign: 'middle', marginRight: 6 }}>
    <line x1="18" y1="20" x2="18" y2="10"/>
    <line x1="12" y1="20" x2="12" y2="4"/>
    <line x1="6" y1="20" x2="6" y2="14"/>
    <line x1="2" y1="20" x2="22" y2="20"/>
  </svg>
)

// ──────────────────────────────────────────────────────────────────
// ENHANCED FEATURE 3: Multi-Currency Support
// Currency symbols and labels used in the dropdown switcher
// ──────────────────────────────────────────────────────────────────
const CURRENCIES = [
  { code: 'USD', symbol: '$', label: 'USD ($)' },
  { code: 'INR', symbol: '₹', label: 'INR (₹)' },
  { code: 'EUR', symbol: '€', label: 'EUR (€)' },
]

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

const emptyForm = { coin_id: '', coin_name: '', quantity: '', price_type: 'p', buy_price: '' }
const emptyRow = { coin_id: '', coin_name: '', quantity: '', price_type: 'p', buy_price: '' }

export default function Portfolio({ token }) {
  const [loading, setLoading] = useState(true)
  const [holdings, setHoldings] = useState([])
  const [total, setTotal] = useState(0)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')

  // Single add
  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState(emptyForm)

  // Multiple add
  const [multiMode, setMultiMode] = useState(false)
  const [rows, setRows] = useState([{ ...emptyRow }])
  const [addingMulti, setAddingMulti] = useState(false)

  // Import / Export
  const [importing, setImporting] = useState(false)
  const [exporting, setExporting] = useState(false)
  const fileRef = useRef(null)

  // ──────────────────────────────────────────────────────────────────
  // ENHANCED FEATURE 2: Fear & Greed Index — state
  // Stores the fetched index data: { value, value_classification }
  // ──────────────────────────────────────────────────────────────────
  const [fearGreed, setFearGreed] = useState(null)

  // ──────────────────────────────────────────────────────────────────
  // ENHANCED FEATURE 3: Multi-Currency Support — state
  // `currency` = selected currency code (USD/INR/EUR)
  // `rates`    = live exchange rates relative to USD
  // ──────────────────────────────────────────────────────────────────
  const [currency, setCurrency] = useState('USD')
  const [rates, setRates] = useState({ USD: 1, INR: 83.5, EUR: 0.92 })

  const hasToken = useMemo(() => Boolean(token), [token])

  async function fetchPortfolio() {
    setLoading(true)
    try {
      const data = await getPortfolio(token)
      setHoldings(data.holdings || [])
      setTotal(data.total_value || 0)
    } catch (err) {
      setError(friendlyError(err.message))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (!hasToken) return
    fetchPortfolio()

    // ──────────────────────────────────────────────────────────────
    // ENHANCED FEATURE 2: Fetch Fear & Greed Index on page load
    // Called once when the portfolio page is opened
    // ──────────────────────────────────────────────────────────────
    getFearGreedIndex()
      .then(setFearGreed)
      .catch(() => {}) // silently fail — non-critical widget

    // ──────────────────────────────────────────────────────────────
    // ENHANCED FEATURE 3: Fetch live exchange rates on page load
    // Falls back to hardcoded rates if the API is unavailable
    // ──────────────────────────────────────────────────────────────
    getExchangeRates()
      .then(r => setRates(prev => ({ ...prev, ...r })))
      .catch(() => {}) // silently fail — fallback rates already set
  }, [token, hasToken])

  // ──────────────────────────────────────────────────────────────────
  // ENHANCED FEATURE 3: Currency conversion helper
  // Multiplies a USD value by the selected currency's exchange rate
  // ──────────────────────────────────────────────────────────────────
  const selectedCurrency = CURRENCIES.find(c => c.code === currency)
  const sym = selectedCurrency.symbol
  const rate = rates[currency] ?? 1
  const convert = (usdVal) => (usdVal * rate).toFixed(2)

  // ──────────────────────────────────────────────────────────────────
  // ENHANCED FEATURE 2: Fear & Greed — determine badge colour
  // Green = Greed (safe/bullish), Red = Fear (bearish/panic)
  // ──────────────────────────────────────────────────────────────────
  function getFngColor(classification) {
    if (!classification) return '#6366f1'
    const c = classification.toLowerCase()
    if (c.includes('extreme greed')) return '#10b981'
    if (c.includes('greed')) return '#22c55e'
    if (c.includes('neutral')) return '#f59e0b'
    if (c.includes('extreme fear')) return '#ef4444'
    if (c.includes('fear')) return '#f97316'
    return '#6366f1'
  }

  // ──────────────────────────────────────────────────────────────────
  // ENHANCED FEATURE 1: Pie Chart data
  // Maps each holding to { name, value } — value = current USD value
  // Recharts uses this to render slices proportional to portfolio %
  // ──────────────────────────────────────────────────────────────────
  const pieData = holdings.map(h => ({
    name: h.coin_name,
    value: parseFloat(h.current_value.toFixed(2)),
  }))

  async function handleAdd(e) {
    e.preventDefault()
    setError('')
    setInfo('')
    setAdding(true)
    try {
      await addHolding(token, {
        ...form,
        quantity: Number(form.quantity),
        buy_price: Number(form.buy_price),
      })
      setInfo('✅ Holding added successfully!')
      setForm(emptyForm)
      await fetchPortfolio()
    } catch (err) {
      setError(friendlyError(err.message))
    } finally {
      setAdding(false)
    }
  }

  function updateRow(idx, field, value) {
    setRows((prev) => prev.map((r, i) => i === idx ? { ...r, [field]: value } : r))
  }

  function addRow() {
    setRows((prev) => [...prev, { ...emptyRow }])
  }

  function removeRow(idx) {
    setRows((prev) => prev.filter((_, i) => i !== idx))
  }

  async function handleAddMultiple(e) {
    e.preventDefault()
    setError('')
    setInfo('')
    setAddingMulti(true)
    try {
      let count = 0
      for (const row of rows) {
        if (!row.coin_id.trim() || !row.quantity || !row.buy_price) continue
        await addHolding(token, {
          coin_id: row.coin_id.trim().toLowerCase(),
          coin_name: row.coin_name.trim() || row.coin_id.trim(),
          quantity: Number(row.quantity),
          price_type: row.price_type,
          buy_price: Number(row.buy_price),
        })
        count++
      }
      setInfo(`✅ ${count} holding(s) added successfully!`)
      setRows([{ ...emptyRow }])
      setMultiMode(false)
      await fetchPortfolio()
    } catch (err) {
      setError(friendlyError(err.message))
    } finally {
      setAddingMulti(false)
    }
  }

  async function handleExport() {
    setError('')
    setExporting(true)
    try {
      await exportPortfolio(token)
      setInfo('✅ Portfolio exported as portfolio.json')
    } catch (err) {
      setError(friendlyError(err.message))
    } finally {
      setExporting(false)
    }
  }

  async function handleImportFile(e) {
    const file = e.target.files[0]
    if (!file) return
    setError('')
    setInfo('')
    setImporting(true)
    try {
      const text = await file.text()
      const json = JSON.parse(text)
      const rawHoldings = Array.isArray(json)
        ? json
        : json.holdings || []

      if (rawHoldings.length === 0) throw new Error('No holdings found in the JSON file.')

      const result = await importPortfolio(token, rawHoldings)
      setInfo(`✅ Imported ${result?.imported ?? rawHoldings.length} holding(s) successfully!`)
      await fetchPortfolio()
    } catch (err) {
      setError(err.message.startsWith('{') ? 'Invalid JSON file.' : friendlyError(err.message))
    } finally {
      setImporting(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  if (!hasToken) return <p>Please log in to see your portfolio.</p>

  return (
    <div>
      {/* ────────────────────────────────────────────────────────────
          ENHANCED FEATURE 2: Fear & Greed Index Widget
          Displayed at the top of the portfolio page.
          Shows a colored badge with score + market sentiment label.
      ──────────────────────────────────────────────────────────── */}
      {fearGreed && (
        <div className="fng-widget">
          <span className="fng-label">Market Sentiment</span>
          <span
            className="fng-badge"
            style={{ background: getFngColor(fearGreed.value_classification) }}
          >
            {fearGreed.value} — {fearGreed.value_classification}
          </span>
          <span className="fng-hint">Crypto Fear &amp; Greed Index</span>
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>My Portfolio</h2>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>

          {/* ──────────────────────────────────────────────────────
              ENHANCED FEATURE 3: Multi-Currency Dropdown
              Lets users switch between USD, INR, and EUR.
              All prices in the table and total update automatically.
          ────────────────────────────────────────────────────── */}
          <select
            className="currency-select"
            value={currency}
            onChange={e => setCurrency(e.target.value)}
            title="Switch display currency"
          >
            {CURRENCIES.map(c => (
              <option key={c.code} value={c.code}>{c.label}</option>
            ))}
          </select>

          <button
            onClick={handleExport}
            disabled={exporting || holdings.length === 0}
            className="button-outline button-inline"
          >
            {exporting ? 'Exporting…' : '⬇ Export JSON'}
          </button>
          <label className="button-outline button-inline" style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 4 }}>
            {importing ? 'Importing…' : '⬆ Import JSON'}
            <input
              ref={fileRef}
              type="file"
              accept=".json,application/json"
              style={{ display: 'none' }}
              onChange={handleImportFile}
              disabled={importing}
            />
          </label>
        </div>
      </div>

      {error && <div className="alert error">{error}</div>}
      {info && <div className="alert success">{info}</div>}

      <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
        {loading ? (
          <p style={{ padding: '32px', textAlign: 'center', color: 'var(--muted)' }}>Loading portfolio…</p>
        ) : holdings.length === 0 ? (
          <p className="empty-state">No holdings yet — add your first one below.</p>
        ) : (
          <>
            {/* Holdings Table — prices shown in selected currency */}
            <table>
              <thead>
                <tr>
                  <th>Coin</th>
                  <th>Qty</th>
                  {/* ENHANCED FEATURE 3: Column headers show selected currency symbol */}
                  <th>Buy Price ({currency})</th>
                  <th>Current ({currency})</th>
                  <th>Value ({currency})</th>
                  <th>P / L</th>
                  <th>P / L %</th>
                </tr>
              </thead>
              <tbody>
                {holdings.map((h) => {
                  const pl = h.profit_loss
                  const plClass = pl >= 0 ? 'profit' : 'loss'
                  return (
                    <tr key={h.coin_id}>
                      <td>
                        <strong>{h.coin_name}</strong>
                        <span style={{ color: 'var(--muted)', fontSize: '0.8em', marginLeft: 6 }}>
                          {h.coin_id}
                        </span>
                      </td>
                      <td>{h.quantity.toFixed(4)}</td>
                      {/* ENHANCED FEATURE 3: All prices converted using live exchange rate */}
                      <td>{sym}{convert(h.buy_price)}</td>
                      <td>{sym}{convert(h.current_price)}</td>
                      <td><strong>{sym}{convert(h.current_value)}</strong></td>
                      <td className={plClass}>
                        {pl >= 0 ? '+' : ''}{sym}{convert(pl)}
                      </td>
                      <td className={plClass}>
                        {h.profit_loss_pct >= 0 ? '+' : ''}
                        {h.profit_loss_pct.toFixed(2)}%
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
            <p className="total" style={{ padding: '12px 20px' }}>
              {/* ENHANCED FEATURE 3: Total also shown in selected currency */}
              Total Portfolio Value: <span style={{ color: 'var(--accent)' }}>{sym}{convert(total)}</span>
            </p>

            {/* ──────────────────────────────────────────────────────
                ENHANCED FEATURE 1: Portfolio Allocation Pie Chart
                Rendered below the holdings table using recharts.
                Each slice = one coin's % share of the portfolio.
                Hover over slices to see coin name and USD value.
            ────────────────────────────────────────────────────── */}
            <div className="pie-section">
              <h3 className="pie-title"><IconChart />Portfolio Allocation</h3>
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie
                    data={pieData}
                    cx="50%"
                    cy="50%"
                    outerRadius={110}
                    dataKey="value"
                    labelLine={false}
                    label={({ name, percent }) =>
                      `${name} ${(percent * 100).toFixed(1)}%`
                    }
                  >
                    {pieData.map((_, index) => (
                      <Cell
                        key={`cell-${index}`}
                        fill={PIE_COLORS[index % PIE_COLORS.length]}
                      />
                    ))}
                  </Pie>
                  <Tooltip
                    formatter={(value) => [`$${value.toFixed(2)}`, 'Value']}
                  />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </>
        )}
      </div>

      {/* Mode toggle */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 12, marginTop: 20 }}>
        <button
          className={!multiMode ? '' : 'button-outline'}
          onClick={() => setMultiMode(false)}
          style={{ flex: 1 }}
        >
          Add Single Holding
        </button>
        <button
          className={multiMode ? '' : 'button-outline'}
          onClick={() => setMultiMode(true)}
          style={{ flex: 1 }}
        >
          Add Multiple Holdings
        </button>
      </div>

      {/* Single add form */}
      {!multiMode && (
        <div className="card">
          <h3>Add Holding</h3>
          <form onSubmit={handleAdd}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
              <div>
                <label>Coin ID <span style={{ color: 'var(--muted)', fontWeight: 400 }}>(e.g. bitcoin)</span></label>
                <input
                  value={form.coin_id}
                  placeholder="bitcoin"
                  onChange={(e) => setForm((s) => ({ ...s, coin_id: e.target.value }))}
                  required
                />
              </div>
              <div>
                <label>Coin Name <span style={{ color: 'var(--muted)', fontWeight: 400 }}>(e.g. Bitcoin)</span></label>
                <input
                  value={form.coin_name}
                  placeholder="Bitcoin"
                  onChange={(e) => setForm((s) => ({ ...s, coin_name: e.target.value }))}
                  required
                />
              </div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10 }}>
              <div>
                <label>Quantity</label>
                <input
                  type="number" step="any" min="0"
                  value={form.quantity} placeholder="0.5"
                  onChange={(e) => setForm((s) => ({ ...s, quantity: e.target.value }))}
                  required
                />
              </div>
              <div>
                <label>Price Type</label>
                <select value={form.price_type} onChange={(e) => setForm((s) => ({ ...s, price_type: e.target.value }))}>
                  <option value="p">Per coin</option>
                  <option value="t">Total paid</option>
                </select>
              </div>
              <div>
                <label>Buy Price ($)</label>
                <input
                  type="number" step="any" min="0"
                  value={form.buy_price} placeholder="40000"
                  onChange={(e) => setForm((s) => ({ ...s, buy_price: e.target.value }))}
                  required
                />
              </div>
            </div>

            <button type="submit" disabled={adding}>
              {adding ? 'Adding…' : '+ Add Holding'}
            </button>
          </form>
        </div>
      )}

      {/* Multiple add form */}
      {multiMode && (
        <div className="card">
          <h3>Add Multiple Holdings</h3>
          <form onSubmit={handleAddMultiple}>
            {rows.map((row, idx) => (
              <div key={idx} style={{ background: 'var(--bg)', borderRadius: 8, padding: '12px 14px', marginBottom: 10, border: '1px solid var(--border)', position: 'relative' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                  <span style={{ fontWeight: 600, fontSize: '0.85rem', color: 'var(--muted)' }}>Holding #{idx + 1}</span>
                  {rows.length > 1 && (
                    <button type="button" onClick={() => removeRow(idx)} className="btn-danger-sm">✕</button>
                  )}
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                  <div>
                    <label>Coin ID</label>
                    <input value={row.coin_id} placeholder="bitcoin" onChange={(e) => updateRow(idx, 'coin_id', e.target.value)} required />
                  </div>
                  <div>
                    <label>Coin Name</label>
                    <input value={row.coin_name} placeholder="Bitcoin" onChange={(e) => updateRow(idx, 'coin_name', e.target.value)} />
                  </div>
                  <div>
                    <label>Quantity</label>
                    <input type="number" step="any" min="0" value={row.quantity} placeholder="0.5" onChange={(e) => updateRow(idx, 'quantity', e.target.value)} required />
                  </div>
                  <div>
                    <label>Price Type</label>
                    <select value={row.price_type} onChange={(e) => updateRow(idx, 'price_type', e.target.value)}>
                      <option value="p">Per coin</option>
                      <option value="t">Total paid</option>
                    </select>
                  </div>
                  <div style={{ gridColumn: '1 / -1' }}>
                    <label>Buy Price ($)</label>
                    <input type="number" step="any" min="0" value={row.buy_price} placeholder="40000" onChange={(e) => updateRow(idx, 'buy_price', e.target.value)} required />
                  </div>
                </div>
              </div>
            ))}

            <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
              <button type="button" onClick={addRow} className="button-outline" style={{ flex: 1 }}>
                + Add Row
              </button>
              <button type="submit" disabled={addingMulti} style={{ flex: 2 }}>
                {addingMulti ? 'Adding…' : `Add ${rows.length} Holding(s)`}
              </button>
            </div>
          </form>
        </div>
      )}
    </div>
  )
}
