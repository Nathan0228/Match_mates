import { useState } from 'react'
import { ArrowRight, X } from 'lucide-react'
import logo from '../../assets/svg/logo.svg'
import userIcon from '../../assets/svg/user.svg'
import lockIcon from '../../assets/svg/lock.svg'
import codeIcon from '../../assets/svg/code.svg'
import ChoiceGroup from '../../components/ChoiceGroup'
import { genderOptions, seekingOptions } from '../../data/identity'
import { request } from '../../lib/api'
import AuthField from './AuthField'
import EmailField, { fullCampusEmail } from './EmailField'

export default function AuthModal({ mode, setMode, onClose, onAuth }) {
  const [emailLocal, setEmailLocal] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [code, setCode] = useState('')
  const [nickname, setNickname] = useState('')
  const [gender, setGender] = useState('')
  const [seeking, setSeeking] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [devCode, setDevCode] = useState('')
  const [notice, setNotice] = useState('')
  const email = fullCampusEmail(emailLocal)

  async function resend() {
    setError('')
    if (!emailLocal || !password) { setError('请先填写邮箱用户名和密码'); return }
    setBusy(true)
    try {
      const data = await request('/resend-code', { method: 'POST', body: JSON.stringify({ email, password }) })
      setDevCode(data.dev_code || '')
      setMode('verify')
    } catch (err) { setError(err.message) } finally { setBusy(false) }
  }

  async function submit(event) {
    event.preventDefault()
    setError('')
    setNotice('')
    if (mode === 'reset' && password !== confirmPassword) { setError('两次输入的密码不一致'); return }
    setBusy(true)
    try {
      if (mode === 'forgot') {
        const data = await request('/password-reset/request', { method: 'POST', body: JSON.stringify({ email }) })
        setDevCode(data.dev_code || '')
        setNotice(data.message)
        setMode('reset')
      } else if (mode === 'reset') {
        const data = await request('/password-reset/confirm', { method: 'POST', body: JSON.stringify({ email, code, password }) })
        setPassword('')
        setConfirmPassword('')
        setCode('')
        setDevCode('')
        setNotice(data.message)
        setMode('login')
      } else if (mode === 'register') {
        const data = await request('/register', { method: 'POST', body: JSON.stringify({ email, password, nickname, gender, seeking }) })
        setDevCode(data.dev_code || '')
        setMode('verify')
      } else if (mode === 'verify') {
        const data = await request('/verify-email', { method: 'POST', body: JSON.stringify({ email, code }) })
        onAuth(data.user)
      } else {
        const data = await request('/login', { method: 'POST', body: JSON.stringify({ email, password }) })
        onAuth(data.user)
      }
    } catch (err) { setError(err.message) } finally { setBusy(false) }
  }

  return (
    <div className="modal-backdrop" onMouseDown={event => event.target === event.currentTarget && onClose()}>
      <div className="auth-modal" role="dialog" aria-modal="true" aria-label="加入见山">
        <button className="modal-close" onClick={onClose} aria-label="关闭"><X size={21} /></button>
        <div className="modal-icon"><img src={logo} alt="" aria-hidden="true" /></div>
        <span className="section-kicker">WELCOME TO JIAN SHAN</span>
        <h2>{mode === 'register' ? '从这里，开始认识。' : mode === 'verify' ? '验证校园邮箱。' : mode === 'forgot' ? '找回密码。' : mode === 'reset' ? '设置新密码。' : '欢迎回来。'}</h2>
        <p>{mode === 'register' ? '仅面向国科大学生。填写信息后，我们会发送一次性验证码。' : mode === 'verify' ? '请输入发送到校园邮箱的 6 位验证码。' : mode === 'forgot' ? '输入注册时使用的校园邮箱，我们会发送重置验证码。' : mode === 'reset' ? '请输入邮箱中的验证码，并设置新密码。' : '继续你的相遇旅程。'}</p>
        <form onSubmit={submit}>
          {mode === 'register' && <AuthField label="昵称" icon={userIcon} required maxLength="24" value={nickname} onChange={event => setNickname(event.target.value)} placeholder="想被怎样称呼？" />}
          {mode === 'register' && <>
            <div className="auth-choice-pair">
              <ChoiceGroup label="我的性别" options={genderOptions} value={gender} onChange={setGender} required />
              <ChoiceGroup label="希望认识" options={seekingOptions} value={seeking} onChange={setSeeking} required />
            </div>
            <p className="auth-fixed-hint">请认真选择，注册后这两项将无法修改。</p>
          </>}
          <EmailField value={emailLocal} onChange={setEmailLocal} disabled={mode === 'verify' || mode === 'reset'} />
          {(mode === 'verify' || mode === 'reset') && <AuthField label="邮箱验证码" icon={codeIcon} required inputMode="numeric" pattern="[0-9]{6}" maxLength="6" value={code} onChange={event => setCode(event.target.value)} placeholder="请输入 6 位数字" />}
          {(mode === 'login' || mode === 'register' || mode === 'reset') && <AuthField label={mode === 'reset' ? '新密码' : '密码'} icon={lockIcon} required type="password" minLength="8" maxLength="128" autoComplete={mode === 'reset' ? 'new-password' : mode === 'register' ? 'new-password' : 'current-password'} value={password} onChange={event => setPassword(event.target.value)} placeholder="至少 8 位" />}
          {mode === 'reset' && <AuthField label="确认新密码" icon={lockIcon} required type="password" minLength="8" maxLength="128" autoComplete="new-password" value={confirmPassword} onChange={event => setConfirmPassword(event.target.value)} placeholder="再次输入新密码" />}
          {devCode && (mode === 'verify' || mode === 'reset') && <div className="dev-code">本地开发验证码：<strong>{devCode}</strong></div>}
          {notice && <div className="form-success" role="status">{notice}</div>}
          {error && <div className="form-error" role="alert">{error}</div>}
          <button className="pill-btn submit-btn" disabled={busy}>{busy ? '请稍候…' : mode === 'register' || mode === 'forgot' ? '发送验证码' : mode === 'verify' ? '完成邮箱验证' : mode === 'reset' ? '重置密码' : '登录'} <ArrowRight size={17} /></button>
        </form>
        {mode === 'login' && <>
          <button className="recover-btn" onClick={() => { setError(''); setNotice(''); setPassword(''); setMode('forgot') }} disabled={busy}>忘记密码？</button>
          <button className="recover-btn" onClick={resend} disabled={busy}>邮箱未验证？用上述邮箱和密码重发验证码</button>
        </>}
        {mode === 'reset' && <button className="recover-btn" onClick={() => { setError(''); setNotice(''); setCode(''); setDevCode(''); setMode('forgot') }} disabled={busy}>重新获取验证码</button>}
        {mode !== 'verify' && <div className="switch-mode">{mode === 'login' ? '还没有账号？' : '已经加入见山？'}<button onClick={() => { setError(''); setNotice(''); setMode(mode === 'login' ? 'register' : 'login') }}>{mode === 'login' ? '立即注册' : '返回登录'}</button></div>}
        <small className="modal-foot">继续即表示了解：审核通过后才能进入匹配池；邮箱仅在双方同意后展示。</small>
      </div>
    </div>
  )
}
