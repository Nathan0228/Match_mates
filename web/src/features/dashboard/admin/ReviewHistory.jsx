import EmptyState from '../../../components/EmptyState'
import formatTime from './formatTime'

export default function ReviewHistory({ history, loading }) {
  return <>
    <p className="admin-history-intro">最近 30 条审核决定。证件文件不会保留在记录中。</p>
    {loading ? <p className="admin-loading">正在加载记录…</p> : history.length ? <div className="admin-history-list">{history.map(item => <div className="admin-history-item" key={item.id}>
      <span className={item.decision === 'approved' ? 'approved' : 'rejected'}>{item.decision === 'approved' ? '已通过' : '已拒绝'}</span>
      <div><strong>{item.nickname}</strong><small>{item.email}</small></div>
      <time dateTime={new Date(item.created_at * 1000).toISOString()}>{formatTime(item.created_at)}</time>
    </div>)}</div> : <EmptyState title="暂无审核记录" text="处理第一份申请后，记录会出现在这里。" />}
  </>
}
