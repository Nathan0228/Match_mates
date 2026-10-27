import { useCallback, useEffect, useRef, useState } from 'react'
import { ArrowLeft, Bell, CheckCheck, X } from 'lucide-react'
import useAutoRefresh from '../hooks/useAutoRefresh'
import { request } from '../lib/api'

function shortTime(unix) {
  return new Date(unix * 1000).toLocaleString('zh-CN', { month: 'numeric', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

export default function NotificationMenu({ open, onToggle, onClose, onNavigate }) {
  const [items, setItems] = useState([])
  const [unread, setUnread] = useState(0)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [selectedAnnouncementId, setSelectedAnnouncementId] = useState(null)
  const loadRequestId = useRef(0)
  const selectedAnnouncement = items.find(item => item.id === selectedAnnouncementId && item.kind === 'announcement')

  const load = useCallback(async () => {
    const requestId = ++loadRequestId.current
    try {
      const data = await request('/notifications')
      if (requestId !== loadRequestId.current) return
      setItems(data.notifications || [])
      setUnread(data.unread_count || 0)
      setError('')
    } catch (err) { if (requestId === loadRequestId.current) setError(err.message) }
    finally { if (requestId === loadRequestId.current) setLoading(false) }
  }, [])

  useAutoRefresh(load)
  useEffect(() => {
    const changed = () => { void load() }
    window.addEventListener('matchmates:data-changed', changed)
    return () => window.removeEventListener('matchmates:data-changed', changed)
  }, [load])
  useEffect(() => { if (open) void load() }, [open, load])
  useEffect(() => { if (!open) setSelectedAnnouncementId(null) }, [open])

  async function openItem(item) {
    if (!item.read_at) {
      try {
        ++loadRequestId.current
        await request(`/notifications/${item.id}/read`, { method: 'POST' })
        setItems(current => current.map(entry => entry.id === item.id ? { ...entry, read_at: Date.now() / 1000 } : entry))
        setUnread(current => Math.max(0, current - 1))
      } catch (err) { setError(err.message); void load(); return }
    }
    if (item.kind === 'announcement') {
      setSelectedAnnouncementId(item.id)
      return
    }
    onClose()
    if (item.target) onNavigate(item.target)
  }

  async function readAll() {
    setBusy(true)
    try {
      ++loadRequestId.current
      await request('/notifications/read-all', { method: 'POST' })
      setItems(current => current.map(item => ({ ...item, read_at: item.read_at || Math.floor(Date.now() / 1000) })))
      setUnread(0)
      setError('')
    } catch (err) { setError(err.message); void load() }
    finally { setBusy(false) }
  }

  return <div className="notification-menu">
    <button className="notification-trigger" type="button" onClick={onToggle} aria-label={`消息提醒${unread ? `，${unread} 条未读` : ''}`} aria-expanded={open} aria-haspopup="dialog">
      <Bell size={22} strokeWidth={1.8} aria-hidden="true" />
      {unread > 0 && <span className="notification-badge" aria-hidden="true">{unread > 99 ? '99+' : unread}</span>}
    </button>
    {open && <section className="notification-panel" role="dialog" aria-label="消息提醒">
      <div className="notification-panel__head"><div><strong>消息提醒</strong><small>{unread ? `${unread} 条未读` : '已读完所有提醒'}</small></div><div>{unread > 0 && <button type="button" className="notification-read-all" onClick={readAll} disabled={busy}><CheckCheck size={15} />全部已读</button>}<button type="button" className="notification-close" onClick={onClose} aria-label="关闭消息"><X size={18} /></button></div></div>
      {error && <p className="notification-error" role="alert">{error}</p>}
      {selectedAnnouncement ? <div className="notification-detail">
        <button type="button" className="notification-detail__back" onClick={() => setSelectedAnnouncementId(null)}><ArrowLeft size={15} aria-hidden="true" />返回消息</button>
        <strong>{selectedAnnouncement.title}</strong>
        <time dateTime={new Date(selectedAnnouncement.created_at * 1000).toISOString()}>{shortTime(selectedAnnouncement.created_at)}</time>
        <p>{selectedAnnouncement.body}</p>
      </div> : <div className="notification-list">{loading ? <p className="notification-empty">正在加载提醒…</p> : items.length ? items.map(item => <button type="button" key={item.id} className={`notification-item${item.read_at ? '' : ' is-unread'}`} onClick={() => openItem(item)}><span className="notification-dot" aria-hidden="true" /><span><strong>{item.title}</strong><span className={`notification-item__body${item.kind === 'announcement' ? ' notification-item__body--preview' : ''}`}>{item.body}</span><time dateTime={new Date(item.created_at * 1000).toISOString()}>{shortTime(item.created_at)}</time></span></button>) : <p className="notification-empty">暂无提醒。有新匹配、公告或审核进展时会显示在这里。</p>}</div>}
    </section>}
  </div>
}
