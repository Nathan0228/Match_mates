import { useState } from 'react'
import Avatar from '../../../components/Avatar'
import formatTime from './formatTime'

const genderName = value => value === 'male' ? '男' : value === 'female' ? '女' : '未填写'
const cardName = value => ({ none: '未提交', pending: '待审核', approved: '已通过', rejected: '需重新提交' })[value] || value

export default function UserDetail({ detail, loading, busy, onStatusChange }) {
  const [confirm, setConfirm] = useState(false)
  const [reason, setReason] = useState('')

  if (loading && !detail) return <div className="admin-review-detail admin-loading">正在加载用户详情…</div>
  if (!detail) return <div className="admin-review-detail admin-loading">选择左侧用户查看详情</div>

  const { user, events } = detail
  const suspended = detail.suspended_at > 0
  const statusName = suspended ? '已停用' : user.status !== 'active' ? '入驻未完成' : user.matching_enabled === false ? '匹配已暂停' : '可参与匹配'
  const preferences = user.preferences || {}

  async function submitStatus() {
    const succeeded = await onStatusChange(!suspended, reason)
    if (succeeded) { setConfirm(false); setReason('') }
  }

  return <div className="admin-review-detail admin-user-detail">
    <div className="admin-user-identity"><Avatar person={user} className="admin-user-avatar" /><div><span className="section-kicker">USER PROFILE</span><h3>{user.nickname}</h3><p>{user.email}</p></div><span className={`admin-user-status ${suspended ? 'suspended' : user.status === 'active' && user.matching_enabled !== false ? 'active' : ''}`}>{statusName}</span></div>
    <dl className="admin-review-fields">
      <div><dt>用户 ID</dt><dd>{user.id}</dd></div><div><dt>注册时间</dt><dd>{formatTime(detail.created_at)}</dd></div>
      <div><dt>性别 / 交友倾向</dt><dd>{genderName(user.gender)} / {genderName(user.seeking)}</dd></div><div><dt>专业 / 校区</dt><dd>{user.major || '未填写'} / {user.campus || '未填写'}</dd></div>
      <div><dt>校园邮箱</dt><dd>{user.email_verified ? '已验证' : '未验证'}</dd></div><div><dt>会费</dt><dd>{user.paid ? '已完成' : '未完成'}</dd></div>
      <div><dt>学生证</dt><dd>{cardName(user.card_status)}</dd></div><div><dt>账号</dt><dd>{suspended ? `停用于 ${formatTime(detail.suspended_at)}` : '正常'}</dd></div>
      <div><dt>匹配展示</dt><dd>{user.matching_enabled === false ? '已暂停' : '已开启'}</dd></div>
    </dl>
    {suspended && <p className="admin-user-reason"><strong>停用原因：</strong>{detail.suspension_reason}</p>}
    <div className="admin-user-profile"><h4>个人介绍</h4><p>{user.intro || '尚未填写'}</p><h4>兴趣与偏好</h4><p>{user.hobbies?.length ? user.hobbies.join(' · ') : '尚未填写兴趣'}{preferences.mbti ? ` · MBTI ${preferences.mbti}` : ''}</p>{preferences.values?.length > 0 && <p>重视：{preferences.values.join(' · ')}</p>}{preferences.daily && <p>周末：{preferences.daily}</p>}{preferences.together && <p>想一起：{preferences.together}</p>}</div>
    {user.is_admin ? <p className="admin-history-intro">管理员账号不可在此停用。</p> : <div className="admin-user-actions">
      {!confirm ? <button type="button" className={suspended ? 'admin-approve' : 'admin-reject'} onClick={() => setConfirm(true)}>{suspended ? '恢复账号' : '停用账号'}</button> : <div className="admin-confirm" role="region" aria-label="确认账号状态"><p>{suspended ? '恢复后，用户可重新登录并继续使用已有资格。' : '停用后，该用户的所有登录会话立即失效，也不会出现在匹配与双向心动中。'}</p><label>{suspended ? '恢复备注（可选）' : '停用原因（至少 4 字）'}<textarea value={reason} onChange={event => setReason(event.target.value)} maxLength={200} rows={3} placeholder={suspended ? '可记录恢复依据' : '请输入停用原因，便于日后追溯'} /></label><div><button type="button" className="admin-cancel" disabled={busy} onClick={() => { setConfirm(false); setReason('') }}>返回</button><button type="button" className={suspended ? 'admin-approve' : 'admin-reject'} disabled={busy || !suspended && reason.trim().length < 4} onClick={submitStatus}>{busy ? '处理中…' : suspended ? '确认恢复' : '确认停用'}</button></div></div>}
    </div>}
    <div className="admin-user-history"><h4>管理记录</h4>{events.length ? events.map((event, index) => <div key={`${event.created_at}-${index}`}><strong>{event.action === 'suspended' ? '停用账号' : '恢复账号'}</strong><small>{formatTime(event.created_at)} · {event.admin_email}</small>{event.reason && <p>{event.reason}</p>}</div>) : <p className="admin-history-intro">暂无账号状态变更记录。</p>}</div>
  </div>
}
