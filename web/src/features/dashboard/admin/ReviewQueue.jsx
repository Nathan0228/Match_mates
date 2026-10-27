import { useState } from 'react'
import { ArrowUpRight, Check, Search, X } from 'lucide-react'
import EmptyState from '../../../components/EmptyState'
import { API_BASE } from '../../../lib/api'
import formatTime from './formatTime'

export default function ReviewQueue({ reviews, loading, onDecide }) {
  const [search, setSearch] = useState('')
  const [selectedId, setSelectedId] = useState(null)
  const [confirmDecision, setConfirmDecision] = useState('')
  const [busy, setBusy] = useState(false)
  const query = search.trim().toLocaleLowerCase()
  const visibleReviews = query ? reviews.filter(person => `${person.nickname} ${person.email} ${person.major} ${person.campus}`.toLocaleLowerCase().includes(query)) : reviews
  const selected = visibleReviews.find(person => person.id === selectedId) || visibleReviews[0]
  const cardUrl = selected ? `${API_BASE}/api/admin/cards/${encodeURIComponent(selected.id)}` : ''

  async function decide() {
    if (!selected || selected.id !== selectedId || !confirmDecision || busy) return
    setBusy(true)
    try {
      if (await onDecide(selected.id, confirmDecision)) {
        setSelectedId(null)
        setConfirmDecision('')
      }
    } finally { setBusy(false) }
  }

  return <>
    <div className="admin-toolbar">
      <p>请核对证件与校园邮箱信息。审核后系统会删除证件文件。</p>
      <label className="admin-search"><Search size={15} /><input value={search} disabled={busy} onChange={event => { setSearch(event.target.value); setConfirmDecision('') }} placeholder="搜索昵称、邮箱或专业" aria-label="搜索待审核申请" /></label>
    </div>
    {loading ? <p className="admin-loading">正在加载申请…</p> : reviews.length === 0 ? <EmptyState title="暂无待审核申请" text="新的学生证申请会显示在这里。" /> : visibleReviews.length === 0 ? <EmptyState title="没有找到符合条件的申请" text="试试其他昵称、邮箱或专业。" /> : <div className="admin-review-layout">
      <div className="admin-review-list" aria-label="待审核申请">
        {visibleReviews.map(person => <button key={person.id} type="button" disabled={busy} className={selected?.id === person.id ? 'selected' : ''} aria-pressed={selected?.id === person.id} onClick={() => { setSelectedId(person.id); setConfirmDecision('') }}>
          <span className="admin-review-avatar">{(person.nickname || '同').slice(0, 1)}</span>
          <span className="admin-review-name"><strong>{person.nickname}</strong><small>{person.email}</small><em>{formatTime(person.submitted_at)}</em></span>
        </button>)}
      </div>
      {selected && <div className="admin-review-detail">
        <div className="admin-review-title"><div><span className="section-kicker">APPLICATION</span><h3>{selected.nickname}的入驻申请</h3></div><span className="admin-pending-tag">待审核</span></div>
        <dl className="admin-review-fields">
          <div><dt>校园邮箱</dt><dd>{selected.email}</dd></div>
          <div><dt>专业 / 研究方向</dt><dd>{selected.major || '未填写'}</dd></div>
          <div><dt>所在校区</dt><dd>{selected.campus || '未填写'}</dd></div>
          <div><dt>提交时间</dt><dd>{formatTime(selected.submitted_at)}</dd></div>
        </dl>
        <div className="admin-proof-heading"><strong>学生证</strong><a href={cardUrl} target="_blank" rel="noreferrer">在新标签查看 <ArrowUpRight size={14} /></a></div>
        <iframe className="admin-proof" title={`${selected.nickname}的学生证`} src={cardUrl} key={selected.id} />
        {confirmDecision && selected.id === selectedId ? <div className="admin-confirm" role="region" aria-label="确认审核决定">
          <p>确认{confirmDecision === 'approved' ? '通过' : '拒绝'} {selected.nickname} 的申请？提交后证件文件会删除，且这次决定不能撤销。</p>
          <div><button type="button" className="admin-cancel" onClick={() => setConfirmDecision('')} disabled={busy}>取消</button><button type="button" className={confirmDecision === 'approved' ? 'admin-approve' : 'admin-reject'} onClick={decide} disabled={busy}>{busy ? '提交中…' : `确认${confirmDecision === 'approved' ? '通过' : '拒绝'}`}</button></div>
        </div> : <div className="admin-actions"><button type="button" className="admin-approve" onClick={() => { setSelectedId(selected.id); setConfirmDecision('approved') }}><Check size={16} /> 通过审核</button><button type="button" className="admin-reject" onClick={() => { setSelectedId(selected.id); setConfirmDecision('rejected') }}><X size={16} /> 拒绝申请</button></div>}
      </div>}
    </div>}
  </>
}
