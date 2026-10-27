import { useState } from 'react'
import { Check, Clock3, RefreshCw, ShieldCheck, UsersRound, X } from 'lucide-react'

export default function AdminOverview({ summary, onRefresh }) {
  const [refreshing, setRefreshing] = useState(false)

  async function handleRefresh() {
    if (refreshing) return
    setRefreshing(true)
    const minimumAnimation = new Promise(resolve => window.setTimeout(resolve, 600))
    try { await onRefresh() }
    finally { await minimumAnimation; setRefreshing(false) }
  }

  return <>
    <div className="admin-heading">
      <div><span className="section-kicker">ADMIN CENTER</span><h2>管理中心</h2><p className="dash-description">查看用户、审核入驻与核对会费，维护安心的交友环境。</p></div>
      <button className={`admin-refresh ${refreshing ? 'is-refreshing' : ''}`} type="button" onClick={handleRefresh} disabled={refreshing} aria-busy={refreshing}><RefreshCw size={15} /> {refreshing ? '刷新中…' : '刷新数据'}</button>
    </div>
    <div className="admin-stats" aria-label="用户和审核概览">
      <div><UsersRound size={19} /><span>注册用户</span><strong>{summary?.total_users ?? '—'}</strong></div>
      <div><Clock3 size={19} /><span>待审核</span><strong>{summary?.pending_reviews ?? '—'}</strong></div>
      <div><Clock3 size={19} /><span>待核款</span><strong>{summary?.pending_payments ?? '—'}</strong></div>
      <div><ShieldCheck size={19} /><span>证件已通过</span><strong>{summary?.approved_cards ?? '—'}</strong></div>
      <div><Check size={19} /><span>已完成会费</span><strong>{summary?.paid_users ?? '—'}</strong></div>
      <div aria-label="付费成功且学生证认证成功的用户数"><UsersRound size={19} /><span>已通过用户</span><strong>{summary?.approved_users ?? '—'}</strong></div>
      <div><X size={19} /><span>需重新提交</span><strong>{summary?.rejected_cards ?? '—'}</strong></div>
      <div><X size={19} /><span>已停用账号</span><strong>{summary?.suspended_users ?? '—'}</strong></div>
    </div>
  </>
}
