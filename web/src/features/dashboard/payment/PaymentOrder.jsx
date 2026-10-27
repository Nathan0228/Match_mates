import { useState } from 'react'
import { API_BASE } from '../../../lib/api'
import { channelName, dateTime, money } from './format'

export default function PaymentOrder({ order, channels, busy, onSubmit, onCancel }) {
  const [reference, setReference] = useState(order.transaction_ref || '')
  const [confirmCancel, setConfirmCancel] = useState(false)
  const currentChannel = channels.find(item => item.channel === order.channel)
  const qr = currentChannel?.version === order.qr_version ? currentChannel : null
  const awaiting = order.status === 'awaiting_payment'
  const rejected = order.status === 'rejected'

  return <div className="payment-order">
    <div className="payment-order-head"><div><span className="section-kicker">PAYMENT ORDER</span><h3>{rejected ? '核款未通过，请核对信息' : order.status === 'submitted' ? '已提交，等待人工核款' : '扫码支付会费'}</h3></div><span className="payment-order-status">{awaiting ? '待付款' : rejected ? '可重新提交' : '待核款'}</span></div>
    <dl className="payment-order-meta">
      <div><dt>订单号</dt><dd>{order.id}</dd></div><div><dt>收款方式</dt><dd>{channelName(order.channel)}</dd></div>
      <div><dt>原价</dt><dd>{money(order.original_cents)}</dd></div><div><dt>优惠</dt><dd>-{money(order.discount_cents)}</dd></div>
      <div className="payment-order-total"><dt>实际应付</dt><dd>{money(order.payable_cents)}</dd></div>
    </dl>
    {order.code_tail && <p className="payment-code-note">已使用折扣码 · 尾号 {order.code_tail}，应付金额已锁定。</p>}
    {rejected && <p className="payment-rejection">未通过原因：{order.review_note || '请联系管理员核实'}。若已付款，请勿重复支付。</p>}
    {order.status === 'submitted' ? <p className="payment-pending">已提交交易单号 {order.transaction_ref}，管理员会核对实际到账。提交记录不代表已付款成功。</p> : <>
      {qr ? <div className="payment-qr"><img src={`${API_BASE}${qr.qr_url}`} alt={`${channelName(order.channel)}收款码`} /><div><strong>请用{channelName(order.channel)}扫码</strong><p>扫码后请核对收款方 <b>{order.payee_name}</b>，手动输入精确金额 <b>{money(order.payable_cents)}</b>，确认无误再转账。</p><small>创建于 {dateTime(order.created_at)}</small></div></div> : <p className="payment-rejection">该订单的收款码已停用或更新。若尚未付款，请取消订单后重新创建；若已付款，请保留交易单号并联系管理员。</p>}
      <label className="payment-reference">付款后填写交易单号 <small>可在微信或支付宝的账单详情中查看</small><input value={reference} onChange={event => setReference(event.target.value)} placeholder="请输入交易单号" maxLength={64} /></label>
      <button type="button" className="pill-btn payment-primary" disabled={busy || reference.trim().length < 6} onClick={() => onSubmit(order.id, reference.trim())}>{busy ? '提交中…' : rejected ? '重新提交核款' : '我已付款，提交核款'}</button>
      <div className="payment-cancel-area">{confirmCancel ? <><span>仅在未付款时取消。若已转账，请先联系管理员。</span><button type="button" disabled={busy} onClick={() => { onCancel(order.id); setConfirmCancel(false) }}>确认取消</button><button type="button" onClick={() => setConfirmCancel(false)}>返回</button></> : <button type="button" disabled={busy} onClick={() => setConfirmCancel(true)}>未付款，取消订单</button>}</div>
    </>}
  </div>
}
