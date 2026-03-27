import { useEffect, useMemo, useRef, useState } from 'react'
import { addHolding, exportPortfolio, importPortfolio, getPortfolio } from '../api'

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

  // Import
  const [importing, setImporting] = useState(false)
  const [exporting, setExporting] = useState(false)
  const fileRef = useRef(null)

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
  }, [token, hasToken])

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
      // Accept either { holdings: [...] } or an array directly or { user_email, holdings: [...] }
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
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0 }}>My Portfolio</h2>
        <div style={{ display: 'flex', gap: 8 }}>
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
            <table>
              <thead>
                <tr>
                  <th>Coin</th>
                  <th>Qty</th>
                  <th>Buy Price</th>
                  <th>Current</th>
                  <th>Value</th>
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
                      <td>${h.buy_price.toFixed(2)}</td>
                      <td>${h.current_price.toFixed(2)}</td>
                      <td><strong>${h.current_value.toFixed(2)}</strong></td>
                      <td className={plClass}>
                        {pl >= 0 ? '+' : ''}${pl.toFixed(2)}
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
              Total Portfolio Value: <span style={{ color: 'var(--accent)' }}>${total.toFixed(2)}</span>
            </p>
          </>
        )}
      </div>

      {/* Mode toggle */}
      <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
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
