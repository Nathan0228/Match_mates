import { useState } from 'react'
import { ArrowLeft, ArrowUpRight, LockKeyhole, LogOut, Settings2, ShieldCheck, Sparkles } from 'lucide-react'
import Checkbox from '../../components/Checkbox'
import { request } from '../../lib/api'
import { readReducedMotion, setReducedMotion } from '../../lib/motion'

export default function SettingsPage({ user, refresh, notify, onBack, onProfile, onPayment, onLogout }) {
  const [matchingBusy, setMatchingBusy] = useState(false)
  const [matchingError, setMatchingError] = useState('')
  const [reducedMotion, setMotionPreference] = useState(readReducedMotion)
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordBusy, setPasswordBusy] = useState(false)
  const [passwordError, setPasswordError] = useState('')

  async function updateMatching(enabled) {
    setMatchingBusy(true)
    setMatchingError('')
    try {
      await request('/settings/matching', { method: 'PUT', body: JSON.stringify({ enabled }) })
      await refresh()
      notify(enabled ? '匹配已恢复' : '匹配已暂停')
    } catch (err) { setMatchingError(err.message) }
    finally { setMatchingBusy(false) }
  }

  async function submitPassword(event) {
    event.preventDefault()
    setPasswordError('')
    if (newPassword !== confirmPassword) { setPasswordError('两次输入的新密码不一致'); return }
    setPasswordBusy(true)
    try {
      await request('/settings/password', { method: 'POST', body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }) })
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      notify('密码已更新，其他设备需要重新登录')
    } catch (err) { setPasswordError(err.message) }
    finally { setPasswordBusy(false) }
  }

  function updateMotion(enabled) {
    setMotionPreference(enabled)
    setReducedMotion(enabled)
  }

  return <div className="settings-page">
    <button className="settings-back" type="button" onClick={onBack}><ArrowLeft size={16} />返回</button>
    <div className="settings-heading"><span className="settings-heading__icon"><Settings2 size={22} /></span><div><span className="section-kicker">YOUR PREFERENCES</span><h2>设置</h2><p>按照自己的节奏使用见山。</p></div></div>

    <section className="settings-section" aria-labelledby="settings-account-title">
      <div className="settings-section__heading"><h3 id="settings-account-title">账户与资料</h3><p>管理你的校园身份和公开介绍</p></div>
      <div className="settings-account-summary">
        <div><span className="section-kicker">YOUR SPACE</span><h4>{user.nickname || '同学'}<span> ☀</span></h4><p>慢慢来，合适的人值得认真遇见。</p></div>
        <span className={`settings-account-status ${user.status === 'active' || user.is_admin ? 'is-on' : ''}`}><i aria-hidden="true" />{user.is_admin ? '管理员' : user.status === 'active' ? '已通过审核' : '完善入驻中'}</span>
      </div>
      <div className="settings-facts"><div><span>校园邮箱</span><strong>{user.email}</strong></div><div><span>认证状态</span><strong>{user.status === 'active' ? '已通过审核' : user.email_verified ? '入驻审核中' : '邮箱待验证'}</strong></div></div>
      <button className="settings-link" type="button" onClick={onProfile}><span>编辑我的介绍<small>头像、昵称、兴趣与相处偏好</small></span><ArrowUpRight size={17} /></button>
      <button className="settings-link" type="button" onClick={onPayment}><span>会费与审核<small>查看付款记录和入驻进度</small></span><ArrowUpRight size={17} /></button>
    </section>

    <section className="settings-section" aria-labelledby="settings-matching-title">
      <div className="settings-section__heading"><h3 id="settings-matching-title"><Sparkles size={17} />匹配与隐私</h3><p>随时暂停，准备好了再回来</p></div>
      <div className="settings-option"><div><Checkbox label="允许展示资料并参与匹配" checked={user.matching_enabled !== false} disabled={matchingBusy} onChange={event => void updateMatching(event.target.checked)} /><p>关闭后，你的资料不再进入新的推荐，也无法浏览或表达新的想认识；已建立的双向心动仍可查看。</p></div><span className={`settings-state ${user.matching_enabled !== false ? 'is-on' : ''}`}>{user.matching_enabled !== false ? '展示中' : '已暂停'}</span></div>
      {matchingError && <p className="settings-error" role="alert">{matchingError}</p>}
      <div className="settings-privacy-note"><ShieldCheck size={18} /><p>校园邮箱仅在双方互相表达想认识后交换；学生证仅用于入驻审核，不会显示在交友资料中。</p></div>
    </section>

    <section className="settings-section" aria-labelledby="settings-security-title">
      <div className="settings-section__heading"><h3 id="settings-security-title"><LockKeyhole size={17} />账户安全</h3><p>修改密码后，其他设备需要重新登录</p></div>
      <form className="settings-password-form" onSubmit={submitPassword}>
        <label>当前密码<input type="password" required autoComplete="current-password" value={currentPassword} onChange={event => setCurrentPassword(event.target.value)} /></label>
        <div className="settings-password-pair">
          <label>新密码<input type="password" required minLength={8} maxLength={128} autoComplete="new-password" value={newPassword} onChange={event => setNewPassword(event.target.value)} placeholder="至少 8 位" /></label>
          <label>确认新密码<input type="password" required minLength={8} maxLength={128} autoComplete="new-password" value={confirmPassword} onChange={event => setConfirmPassword(event.target.value)} /></label>
        </div>
        {passwordError && <p className="settings-error" role="alert">{passwordError}</p>}
        <button className="settings-primary" type="submit" disabled={passwordBusy}>{passwordBusy ? '更新中…' : '更新密码'}</button>
      </form>
    </section>

    <section className="settings-section" aria-labelledby="settings-display-title">
      <div className="settings-section__heading"><h3 id="settings-display-title">显示偏好</h3><p>仅影响当前浏览器</p></div>
      <div className="settings-option"><div><Checkbox label="减少页面动态效果" checked={reducedMotion} onChange={event => updateMotion(event.target.checked)} /><p>减少滚动出现、菜单与按钮的动画；系统的减少动态效果设置也会生效。</p></div></div>
    </section>

    <div className="settings-footer"><button type="button" onClick={onLogout}><LogOut size={16} />退出登录</button></div>
  </div>
}
