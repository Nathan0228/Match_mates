import { useCallback, useEffect, useState } from 'react'
import { request } from '../../lib/api'
import useAutoRefresh from '../../hooks/useAutoRefresh'
import AdminOverview from './admin/AdminOverview'
import ReviewQueue from './admin/ReviewQueue'
import ReviewHistory from './admin/ReviewHistory'
import PaymentAdmin from './admin/PaymentAdmin'
import PromoAdmin from './admin/PromoAdmin'
import UserDirectory from './admin/UserDirectory'
import AnnouncementAdmin from './admin/AnnouncementAdmin'

export default function AdminReviews({ notify, viewRequest }) {
  const [summary, setSummary] = useState(null)
  const [reviews, setReviews] = useState([])
  const [history, setHistory] = useState([])
  const [view, setView] = useState(() => viewRequest?.view || 'queue')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => { if (viewRequest?.id) setView(viewRequest.view) }, [viewRequest])

  const refreshAdmin = useCallback(async () => {
    try {
      const [overview, pending, recent] = await Promise.all([
        request('/admin/summary'), request('/admin/reviews'), request('/admin/history'),
      ])
      setSummary(overview)
      setReviews(pending.reviews || [])
      setHistory(recent.history || [])
      setError('')
    } catch (err) { setError(err.message) }
    finally { setLoading(false) }
  }, [])

  useAutoRefresh(refreshAdmin)

  async function decide(id, decision) {
    setError('')
    try {
      const result = await request(`/admin/reviews/${encodeURIComponent(id)}`, { method: 'POST', body: JSON.stringify({ decision }) })
      notify(result.file_deleted ? (decision === 'approved' ? '审核已通过，证件文件已删除' : '申请已拒绝，证件文件已删除') : '审核已完成，但证件文件清理失败，请检查服务器日志')
      setReviews(items => items.filter(person => person.id !== id))
      setSummary(current => current && {
        ...current,
        pending_reviews: Math.max(0, current.pending_reviews - 1),
        [decision === 'approved' ? 'approved_cards' : 'rejected_cards']: current[decision === 'approved' ? 'approved_cards' : 'rejected_cards'] + 1,
      })
      await refreshAdmin()
      return true
    } catch (err) {
      if (err.status === 404 || err.status === 409) await refreshAdmin()
      setError(err.message)
      return false
    }
  }

  return <section className="admin-page">
    <AdminOverview summary={summary} onRefresh={refreshAdmin} />
    {error && <div className="form-error" role="alert">{error}</div>}
    <div className="admin-tabs" role="tablist" aria-label="管理内容">
      <button type="button" role="tab" aria-selected={view === 'queue'} className={view === 'queue' ? 'selected' : ''} onClick={() => setView('queue')}>待审核 <span>{summary?.pending_reviews ?? reviews.length}</span></button>
      <button type="button" role="tab" aria-selected={view === 'payments'} className={view === 'payments' ? 'selected' : ''} onClick={() => setView('payments')}>会费核款 <span>{summary?.pending_payments ?? 0}</span></button>
      <button type="button" role="tab" aria-selected={view === 'promos'} className={view === 'promos' ? 'selected' : ''} onClick={() => setView('promos')}>优惠代码</button>
      <button type="button" role="tab" aria-selected={view === 'users'} className={view === 'users' ? 'selected' : ''} onClick={() => setView('users')}>用户管理</button>
      <button type="button" role="tab" aria-selected={view === 'announcements'} className={view === 'announcements' ? 'selected' : ''} onClick={() => setView('announcements')}>消息通知</button>
      <button type="button" role="tab" aria-selected={view === 'history'} className={view === 'history' ? 'selected' : ''} onClick={() => setView('history')}>审核记录</button>
    </div>
    {view === 'queue' && <ReviewQueue reviews={reviews} loading={loading} onDecide={decide} />}
    {view === 'payments' && <PaymentAdmin notify={notify} onChanged={refreshAdmin} />}
    {view === 'promos' && <PromoAdmin notify={notify} />}
    {view === 'users' && <UserDirectory notify={notify} onChanged={refreshAdmin} />}
    {view === 'announcements' && <AnnouncementAdmin notify={notify} />}
    {view === 'history' && <ReviewHistory history={history} loading={loading} />}
  </section>
}
