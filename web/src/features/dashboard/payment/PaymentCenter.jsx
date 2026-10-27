import { useCallback, useState } from 'react'
import { CheckCircle2, TicketPercent } from 'lucide-react'
import { request } from '../../../lib/api'
import useAutoRefresh from '../../../hooks/useAutoRefresh'
import PaymentOrder from './PaymentOrder'
import { channelName, dateTime, money } from './format'

export default function PaymentCenter({ user, refresh, notify }) {
  const [data, setData] = useState(null)
  const [code, setCode] = useState('')
  const [quote, setQuote] = useState(null)
  const [channel, setChannel] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    try { setData(await request('/payment/status')); setError('') }
    catch (err) { setError(err.message) }
  }, [])
  useAutoRefresh(load)

  async function applyCode() {
    setBusy(true); setError(''); setQuote(null)
    try { setQuote(await request('/payment/quote', { method: 'POST', body: JSON.stringify({ code }) })) }
    catch (err) { setError(err.message) }
    finally { setBusy(false) }
  }

  async function act(path, body, success, updateUser = false) {
    setBusy(true); setError('')
    try {
      await request(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined })
      await load()
      if (updateUser) await refresh()
      notify(success)
      return true
    } catch (err) { setError(err.message); return false }
    finally { setBusy(false) }
  }

  const paid = data?.paid || user.paid
  const order = data?.order
  const activeOrder = order && ['awaiting_payment', 'submitted', 'rejected'].includes(order.status)
  const channels = data?.channels || []

  return <section className="payment-center">
    <span className="section-kicker">MEMBERSHIP</span><h2>会费与兑换</h2>
    <p className="dash-description">一次性会费 {money(data?.original_cents ?? 990)}。可使用兑换码，或用折扣码减免部分金额。</p>
    {error && <div className="form-error" role="alert">{error}</div>}
    {paid ? <div className="payment-complete"><CheckCircle2 size={27} /><div><strong>会费已完成</strong><p>可以继续完成学生证审核与个人资料。</p></div></div> : activeOrder ? <PaymentOrder key={order.id} order={order} channels={channels} busy={busy} onSubmit={(id, transaction_ref) => act(`/payment/orders/${id}/submit`, { transaction_ref }, '已提交交易单号，等待管理员核对')} onCancel={id => act(`/payment/orders/${id}/cancel`, null, '订单已取消')} /> : <>
      <div className="payment-code-entry"><TicketPercent size={21} /><div><strong>有兑换码或折扣码？</strong><p>兑换码免除全部会费；折扣码按支付比例减免。</p></div></div>
      <div className="payment-code-form"><input value={code} onChange={event => { setCode(event.target.value); setQuote(null) }} placeholder="输入代码" maxLength={80} /><button type="button" disabled={busy || !code.trim()} onClick={applyCode}>{busy ? '验证中…' : '验证代码'}</button></div>
      {quote && <div className="payment-quote"><strong>{quote.kind === 'exchange' ? '兑换码可免除全部会费' : `支付比例 ${quote.pay_percent}% · 实付 ${money(quote.payable_cents)}`}</strong><span>原价 {money(quote.original_cents)}，减免 {money(quote.discount_cents)}</span>{quote.kind === 'exchange' && <button type="button" className="pill-btn payment-primary" disabled={busy} onClick={() => act('/payment/redeem', { code }, '兑换成功，会费已完成', true)}>确认兑换</button>}</div>}
      {quote?.kind !== 'exchange' && <div className="payment-create"><h3>选择收款方式</h3><p>创建订单后显示收款码与准确应付金额。付款后还需提交交易单号，由管理员核对实际到账。</p>
        {channels.length ? <div className="payment-channels">{channels.map(item => <label key={item.channel} className={channel === item.channel ? 'selected' : ''}><input type="radio" name="payment-channel" checked={channel === item.channel} onChange={() => setChannel(item.channel)} />{channelName(item.channel)}</label>)}</div> : <p className="payment-rejection">管理员尚未配置可用收款码，请稍后再试。</p>}
        <div className="payment-create-total"><span>应付金额</span><strong>{money(quote?.payable_cents ?? 990)}</strong></div>
        <button type="button" className="pill-btn payment-primary" disabled={busy || !channel || Boolean(code.trim() && !quote)} onClick={() => act('/payment/orders', { channel, code: quote?.kind === 'discount' ? code : '' }, '订单已创建，请按显示金额扫码付款')}>创建付款订单</button>
      </div>}
      {order?.status === 'cancelled' && <p className="payment-old-order">上一个订单已取消，可重新选择收款方式。</p>}
    </>}
    {data?.uses?.length > 0 && <div className="payment-uses"><h3>代码使用记录</h3>{data.uses.map((item, index) => <div key={`${item.created_at}-${index}`}><span>{item.kind === 'exchange' ? '兑换码' : '折扣码'} · 尾号 {item.code_tail}</span><strong>减免 {money(item.discount_cents)}</strong><small>{dateTime(item.created_at)}</small></div>)}</div>}
    <p className="payment-safety">收款码付款不会自动开通资格。管理员会对照收款账户核对交易单号和金额；若核款未通过，请按页面说明处理，避免重复付款。</p>
  </section>
}
