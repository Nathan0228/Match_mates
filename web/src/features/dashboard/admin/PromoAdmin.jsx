import { useCallback, useState } from 'react'
import { Copy } from 'lucide-react'
import { request } from '../../../lib/api'
import useAutoRefresh from '../../../hooks/useAutoRefresh'
import { dateTime, money } from '../payment/format'

function localInput(date) {
  const pad = value => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function codeStatus(item, now) {
  if (item.kind === 'exchange' && item.max_uses > 0 && item.used_count >= item.max_uses) {
    return { label: '已用尽', className: 'used', finished: true }
  }
  if (now > item.valid_until) return { label: '已过期', className: 'expired', finished: true }
  return { label: item.active ? '启用中' : '已停用', className: item.active ? 'active' : '', finished: false }
}

export default function PromoAdmin({ notify }) {
  const [codes, setCodes] = useState([])
  const [uses, setUses] = useState([])
  const [kind, setKind] = useState('exchange')
  const [payPercent, setPayPercent] = useState(80)
  const [maxUses, setMaxUses] = useState(1)
  const [start, setStart] = useState(() => localInput(new Date()))
  const [end, setEnd] = useState(() => localInput(new Date(Date.now() + 30 * 86400000)))
  const [issuedCode, setIssuedCode] = useState('')
  const [copyingId, setCopyingId] = useState('')
  const [manualCopy, setManualCopy] = useState(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    try {
      const [codeData, useData] = await Promise.all([request('/admin/promo-codes'), request('/admin/promo-uses')])
      setCodes(codeData.codes || [])
      setUses(useData.uses || [])
      setError('')
    } catch (err) { setError(err.message) }
  }, [])
  useAutoRefresh(load)

  async function create(event) {
    event.preventDefault()
    setBusy(true); setError(''); setIssuedCode('')
    try {
      const result = await request('/admin/promo-codes', { method: 'POST', body: JSON.stringify({ kind, pay_percent: Number(payPercent), max_uses: Number(maxUses), valid_from: Math.floor(new Date(start).getTime() / 1000), valid_until: Math.floor(new Date(end).getTime() / 1000) }) })
      setIssuedCode(result.code)
      await load()
      notify('代码已生成，可在代码管理中复制')
    } catch (err) { setError(err.message) }
    finally { setBusy(false) }
  }

  async function toggle(id, active) {
    setBusy(true); setError('')
    try {
      await request(`/admin/promo-codes/${encodeURIComponent(id)}/status`, { method: 'POST', body: JSON.stringify({ active }) })
      await load()
      notify(active ? '代码已启用' : '代码已停用')
    } catch (err) { setError(err.message) }
    finally { setBusy(false) }
  }

  async function copyText(code, id = '') {
    try {
      if (!navigator.clipboard?.writeText) throw new Error('clipboard unavailable')
      await navigator.clipboard.writeText(code)
      setManualCopy(null)
      notify('代码已复制')
    } catch {
      if (id) setManualCopy({ id, code })
      notify('自动复制失败，请手动选择代码')
    }
  }

  async function copyExisting(id) {
    setCopyingId(id)
    setManualCopy(null)
    try {
      const result = await request(`/admin/promo-codes/${encodeURIComponent(id)}/code`)
      await copyText(result.code, id)
    } catch (err) { notify(err.message) }
    finally { setCopyingId('') }
  }

  const now = Math.floor(Date.now() / 1000)

  return <div className="promo-admin">
    <p className="admin-history-intro">兑换码免除全部会费，可限制总使用次数。折扣码按支付比例计算实付金额；两类代码均可设置时间范围并随时停用。</p>
    {error && <div className="form-error" role="alert">{error}</div>}
    <form className="promo-form" onSubmit={create}>
      <h3>生成新代码</h3>
      <div className="field-pair"><label>代码类型<select value={kind} onChange={event => setKind(event.target.value)}><option value="exchange">兑换码 · 免全部会费</option><option value="discount">折扣码 · 按比例支付</option></select></label>
        {kind === 'exchange' ? <label>最多使用次数<input type="number" min="1" max="100000" required value={maxUses} onChange={event => setMaxUses(event.target.value)} /></label> : <label>支付比例（%）<input type="number" min="1" max="99" required value={payPercent} onChange={event => setPayPercent(event.target.value)} /><small>80% 即八折，9.9 元实付 7.92 元</small></label>}
      </div>
      <div className="field-pair"><label>生效时间<input type="datetime-local" required value={start} onChange={event => setStart(event.target.value)} /></label><label>结束时间<input type="datetime-local" required value={end} onChange={event => setEnd(event.target.value)} /></label></div>
      <button type="submit" className="pill-btn" disabled={busy}>{busy ? '生成中…' : '生成代码'}</button>
      {issuedCode && <div className="promo-issued" role="status"><span>新代码已生成，也可在下方代码管理中再次复制</span><strong>{issuedCode}</strong><button type="button" onClick={() => copyText(issuedCode)}><Copy size={14} /> 复制代码</button></div>}
    </form>
    <section className="promo-section"><h3>代码管理</h3>{codes.length ? <div className="promo-list">{codes.map(item => {
      const status = codeStatus(item, now)
      return <div className="promo-item" key={item.id}>
      <div className="promo-item__info"><strong>{item.kind === 'exchange' ? '兑换码' : `${item.pay_percent}% 支付折扣码`} · 尾号 {item.code_tail}</strong><small>{dateTime(item.valid_from)} — {dateTime(item.valid_until)}</small><small>已使用 {item.used_count} 次{item.kind === 'exchange' ? ` / 共 ${item.max_uses} 次` : ''}</small></div>
      <span className={status.className}>{status.label}</span>
      <div className="promo-item__actions">{item.copy_available ? <button type="button" disabled={copyingId === item.id} onClick={() => copyExisting(item.id)}><Copy size={13} />{copyingId === item.id ? '复制中…' : '复制代码'}</button> : <span className="promo-item__legacy" title="旧代码只保存摘要，无法恢复完整内容">旧码无法复制</span>}{!status.finished && <button type="button" disabled={busy} onClick={() => toggle(item.id, !item.active)}>{item.active ? '停用' : '启用'}</button>}</div>
      {manualCopy?.id === item.id && <div className="promo-manual-copy"><small>请手动选择下面的完整代码复制</small><code>{manualCopy.code}</code><button type="button" onClick={() => setManualCopy(null)}>隐藏代码</button></div>}
    </div> })}</div> : <p className="admin-history-intro">还没有创建代码。</p>}</section>
    <section className="promo-section"><h3>最近使用记录</h3>{uses.length ? <div className="promo-use-list">{uses.map(item => <div key={item.id}><strong>{item.nickname} · {item.kind === 'exchange' ? '兑换' : '折扣'} {money(item.discount_cents)}</strong><small>{item.email} · 尾号 {item.code_tail} · {dateTime(item.created_at)}</small><small>实付 {money(item.payable_cents)}{item.order_id ? ` · 订单 ${item.order_id}` : ''}</small></div>)}</div> : <p className="admin-history-intro">暂无代码使用记录。</p>}</section>
  </div>
}
