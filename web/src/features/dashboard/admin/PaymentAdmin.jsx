import { useCallback, useState } from 'react'
import EmptyState from '../../../components/EmptyState'
import Checkbox from '../../../components/Checkbox'
import { request } from '../../../lib/api'
import useAutoRefresh from '../../../hooks/useAutoRefresh'
import { channelName, dateTime, money } from '../payment/format'
import QRSettings from './QRSettings'

export default function PaymentAdmin({ notify, onChanged }) {
  const [orders, setOrders] = useState({ pending: [], approved: [], rejected: [] })
  const [qrCodes, setQRCodes] = useState([])
  const [view, setView] = useState('pending')
  const [selectedId, setSelectedId] = useState(null)
  const [decision, setDecision] = useState('')
  const [verified, setVerified] = useState(false)
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    try {
      const [payments, qr] = await Promise.all([request('/admin/payment/orders'), request('/admin/payment/qr')])
      setOrders(payments)
      setQRCodes(qr.codes || [])
      setError('')
    } catch (err) { setError(err.message) }
  }, [])
  useAutoRefresh(load)

  const selected = orders.pending.find(item => item.id === selectedId) || orders.pending[0]
  const recent = [...orders.approved, ...orders.rejected].sort((a, b) => b.reviewed_at - a.reviewed_at).slice(0, 30)

  async function decide() {
    if (!selected || selected.id !== selectedId || !decision || busy) return
    setBusy(true); setError('')
    try {
      await request(`/admin/payment/orders/${encodeURIComponent(selected.id)}`, { method: 'POST', body: JSON.stringify({ decision, verified, note }) })
      notify(decision === 'approved' ? '已确认到账，会费资格已开通' : '已退回核款申请，用户可修正交易单号')
      setDecision(''); setVerified(false); setNote(''); setSelectedId(null)
      await load(); await onChanged()
    } catch (err) { setError(err.message) }
    finally { setBusy(false) }
  }

  return <div className="payment-admin">
    <div className="payment-admin-tabs"><button type="button" className={view === 'pending' ? 'selected' : ''} onClick={() => setView('pending')}>待核款 ({orders.pending.length})</button><button type="button" className={view === 'qr' ? 'selected' : ''} onClick={() => setView('qr')}>收款码设置</button><button type="button" className={view === 'recent' ? 'selected' : ''} onClick={() => setView('recent')}>核款记录</button></div>
    {error && <div className="form-error" role="alert">{error}</div>}
    {view === 'qr' && <QRSettings codes={qrCodes} onChanged={load} notify={notify} />}
    {view === 'pending' && (orders.pending.length ? <div className="admin-review-layout">
      <div className="admin-review-list">{orders.pending.map(item => <button type="button" key={item.id} className={selected?.id === item.id ? 'selected' : ''} onClick={() => { setSelectedId(item.id); setDecision(''); setVerified(false); setNote('') }} disabled={busy}><span className="admin-review-name"><strong>{item.nickname} · {money(item.payable_cents)}</strong><small>{item.email}</small><em>{channelName(item.channel)} · {dateTime(item.submitted_at)}</em></span></button>)}</div>
      {selected && <div className="admin-review-detail"><div className="admin-review-title"><div><span className="section-kicker">PAYMENT REVIEW</span><h3>核对实际到账</h3></div><span className="admin-pending-tag">待核款</span></div>
        <dl className="admin-review-fields"><div><dt>用户</dt><dd>{selected.nickname} · {selected.email}</dd></div><div><dt>订单号</dt><dd>{selected.id}</dd></div><div><dt>付款渠道</dt><dd>{channelName(selected.channel)}</dd></div><div><dt>下单时的收款方</dt><dd>{selected.payee_name}</dd></div><div><dt>用户填写的交易单号</dt><dd>{selected.transaction_ref}</dd></div><div><dt>原价 / 减免</dt><dd>{money(selected.original_cents)} / {money(selected.discount_cents)}</dd></div><div><dt>须核实的到账金额</dt><dd className="payment-admin-amount">{money(selected.payable_cents)}</dd></div></dl>
        {selected.code_tail && <p className="payment-code-note">折扣码尾号 {selected.code_tail}；订单金额已锁定。</p>}
        {selected.duplicate_ref && <p className="payment-rejection">该交易单号也出现在其他订单，请重点核对，避免重复确认。</p>}
        <p className="payment-admin-warning">用户提交的单号只能作为线索。请在自己的收款账户中核对渠道、交易单号与实际到账金额。</p>
        {decision ? <div className="admin-confirm" role="region" aria-label="确认核款决定"><p>确认{decision === 'approved' ? '通过并开通会费资格' : '退回该核款申请'}？</p>
          {decision === 'approved' ? <Checkbox className="payment-verified" label="我已在收款账户核对这笔交易的单号与金额" checked={verified} onChange={event => setVerified(event.target.checked)} /> : <label className="payment-review-note">未通过原因<textarea value={note} onChange={event => setNote(event.target.value)} maxLength={200} rows={3} placeholder="例如：未查到对应交易，请核对交易单号" /></label>}
          <div><button type="button" className="admin-cancel" onClick={() => setDecision('')} disabled={busy}>返回</button><button type="button" className={decision === 'approved' ? 'admin-approve' : 'admin-reject'} disabled={busy || decision === 'approved' && !verified || decision === 'rejected' && note.trim().length < 4} onClick={decide}>{busy ? '提交中…' : '确认处理'}</button></div>
        </div> : <div className="admin-actions"><button type="button" className="admin-approve" onClick={() => { setSelectedId(selected.id); setDecision('approved') }}>核实通过</button><button type="button" className="admin-reject" onClick={() => { setSelectedId(selected.id); setDecision('rejected') }}>退回核款</button></div>}
      </div>}
    </div> : <EmptyState title="暂无待核款订单" text="用户扫码付款并提交交易单号后，订单会显示在这里。" />)}
    {view === 'recent' && (recent.length ? <div className="admin-history-list">{recent.map(item => <div className="admin-history-item" key={item.id}><span className={item.status === 'approved' ? 'approved' : 'rejected'}>{item.status === 'approved' ? '已通过' : '已退回'}</span><div><strong>{item.nickname} · {money(item.payable_cents)}</strong><small>{item.email} · {channelName(item.channel)} · {item.transaction_ref}</small>{item.review_note && <small>原因：{item.review_note}</small>}</div><time>{dateTime(item.reviewed_at)}</time></div>)}</div> : <EmptyState title="暂无核款记录" text="处理后的订单会显示在这里。" />)}
  </div>
}
