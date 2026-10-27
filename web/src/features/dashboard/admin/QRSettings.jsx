import { useState } from 'react'
import { API_BASE, request } from '../../../lib/api'
import { channelName, dateTime } from '../payment/format'

export default function QRSettings({ codes, onChanged, notify }) {
  const [files, setFiles] = useState({})
  const [names, setNames] = useState({})
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')

  async function upload(channel) {
    const file = files[channel]
    if (!file) return
    setBusy(channel); setError('')
    const body = new FormData()
    body.append('qr', file)
    body.append('payee_name', (names[channel] ?? codes.find(item => item.channel === channel)?.payee_name ?? '').trim())
    try {
      await request(`/admin/payment/qr/${channel}`, { method: 'POST', body })
      setFiles(current => ({ ...current, [channel]: null }))
      await onChanged()
      notify(`${channelName(channel)}收款码已更新`)
    } catch (err) { setError(err.message) }
    finally { setBusy('') }
  }

  async function saveName(channel) {
    setBusy(channel); setError('')
    try {
      await request(`/admin/payment/qr/${channel}/settings`, { method: 'POST', body: JSON.stringify({ payee_name: (names[channel] || '').trim() }) })
      await onChanged()
      notify('收款方名称已更新')
    } catch (err) { setError(err.message) }
    finally { setBusy('') }
  }

  async function toggle(channel, active) {
    setBusy(channel); setError('')
    try {
      await request(`/admin/payment/qr/${channel}/status`, { method: 'POST', body: JSON.stringify({ active }) })
      await onChanged()
      notify(active ? '收款方式已启用' : '收款方式已停用')
    } catch (err) { setError(err.message) }
    finally { setBusy('') }
  }

  return <div className="payment-admin-section">
    <h3>收款码设置</h3><p>请上传由你控制的微信或支付宝收款码。停用后用户不能再用该渠道创建新订单。</p>
    {error && <div className="form-error" role="alert">{error}</div>}
    <div className="admin-qr-grid">{['wechat', 'alipay'].map(channel => {
      const item = codes.find(code => code.channel === channel)
      return <div className="admin-qr-card" key={channel}>
        <div className="admin-qr-top"><strong>{channelName(channel)}</strong><span>{item ? item.active ? '使用中' : '已停用' : '未设置'}</span></div>
        {item ? <img src={`${API_BASE}${item.qr_url}?v=${item.updated_at}`} alt={`${channelName(channel)}收款码预览`} /> : <div className="admin-qr-placeholder">等待上传</div>}
        <small>{item ? `更新于 ${dateTime(item.updated_at)}` : '仅支持 PNG 或 JPG，最多 2 MB'}</small>
        <label className="admin-qr-name">收款方名称<input value={names[channel] ?? item?.payee_name ?? ''} onChange={event => setNames(current => ({ ...current, [channel]: event.target.value }))} placeholder="用户扫码后应看到的收款方" maxLength={60} /></label>
        {item && names[channel] !== undefined && names[channel].trim() !== item.payee_name && <button type="button" disabled={busy === channel} onClick={() => saveName(channel)}>保存收款方名称</button>}
        <label className="admin-qr-pick">选择图片<input type="file" accept="image/png,image/jpeg" onChange={event => setFiles(current => ({ ...current, [channel]: event.target.files?.[0] || null }))} /></label>
        {files[channel] && <button type="button" disabled={busy === channel} onClick={() => upload(channel)}>{busy === channel ? '上传中…' : '上传新收款码'}</button>}
        {item && <button type="button" className={item.active ? 'pause' : ''} disabled={busy === channel} onClick={() => toggle(channel, !item.active)}>{item.active ? '停用收款码' : '启用收款码'}</button>}
      </div>
    })}</div>
  </div>
}
