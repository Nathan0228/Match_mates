import { useCallback, useState } from 'react'
import { Megaphone, Send } from 'lucide-react'
import useAutoRefresh from '../../../hooks/useAutoRefresh'
import { request } from '../../../lib/api'
import EmptyState from '../../../components/EmptyState'
import formatTime from './formatTime'

export default function AnnouncementAdmin({ notify }) {
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [confirm, setConfirm] = useState(false)
  const [busy, setBusy] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [items, setItems] = useState([])

  const load = useCallback(async () => {
    try {
      const data = await request('/admin/announcements')
      setItems(data.announcements || [])
      setError('')
    } catch (err) { setError(err.message) }
    finally { setLoading(false) }
  }, [])

  useAutoRefresh(load)

  function preview(event) {
    event.preventDefault()
    if (title.trim().length < 2 || !body.trim()) {
      setError('请填写至少 2 字的标题和消息内容')
      return
    }
    setError('')
    setConfirm(true)
  }

  async function publish() {
    if (busy) return
    setBusy(true)
    setError('')
    try {
      const item = await request('/admin/announcements', { method: 'POST', body: JSON.stringify({ title, body }) })
      setItems(current => [item, ...current].slice(0, 30))
      setTitle('')
      setBody('')
      setConfirm(false)
      notify(`消息已发送给 ${item.recipient_count} 位用户`)
    } catch (err) { setError(err.message) }
    finally { setBusy(false) }
  }

  return <div className="admin-announcements">
    <div className="admin-announcements__intro"><Megaphone size={19} aria-hidden="true" /><div><h3>发布消息通知</h3><p>消息将发送到当前所有已注册用户的消息提醒中，管理员账号除外。</p></div></div>
    <form className="admin-announcement-form" onSubmit={preview}>
      <label>标题<input value={title} onChange={event => { setTitle(event.target.value); setConfirm(false) }} minLength={2} maxLength={60} required placeholder="例如：网站服务公告" disabled={busy} /></label>
      <label>消息内容<textarea value={body} onChange={event => { setBody(event.target.value); setConfirm(false) }} maxLength={1000} required rows={5} placeholder="清晰写明需要告知同学的内容" disabled={busy} /></label>
      <div className="admin-announcement-form__foot"><span>{body.length} / 1000 字</span><button type="submit" disabled={busy}>预览消息</button></div>
    </form>
    {error && <p className="form-error" role="alert">{error}</p>}
    {confirm && <div className="admin-announcement-preview" role="region" aria-label="发布前预览">
      <span className="section-kicker">发布前预览</span>
      <strong>{title.trim()}</strong>
      <p>{body.trim()}</p>
      <div><button type="button" className="admin-cancel" onClick={() => setConfirm(false)} disabled={busy}>返回编辑</button><button type="button" className="admin-approve" onClick={publish} disabled={busy}><Send size={15} aria-hidden="true" />{busy ? '发布中…' : '确认发布'}</button></div>
    </div>}
    <div className="admin-announcements__history"><h3>发布记录</h3><p>最近 30 条消息</p></div>
    {loading ? <div className="admin-loading">正在加载发布记录…</div> : items.length ? <div className="admin-announcement-list">{items.map(item => <article key={item.id}>
      <div><strong>{item.title}</strong><time dateTime={new Date(item.created_at * 1000).toISOString()}>{formatTime(item.created_at)} · 发送给 {item.recipient_count} 位用户</time></div>
      <p>{item.body}</p>
    </article>)}</div> : <EmptyState title="暂无发布记录" text="发布后的消息会留存在这里。" />}
  </div>
}
