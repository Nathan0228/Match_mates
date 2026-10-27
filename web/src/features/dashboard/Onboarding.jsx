import { useState } from 'react'
import { ArrowRight, Check, MessageCircle, ShieldCheck, UploadCloud } from 'lucide-react'
import { request } from '../../lib/api'
import Reveal from '../../components/Reveal'

export default function Onboarding({ user, refresh, notify, onProfile, onPayment }) {
  const [cardFile, setCardFile] = useState(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const profileDone = Boolean(user.gender && user.seeking && user.intro?.length >= 20 && user.hobbies?.length >= 2)
  const completed = [user.email_verified, user.paid, user.card_status === 'approved', profileDone].filter(Boolean).length

  async function uploadCard() {
    if (!cardFile) return
    setBusy(true)
    setError('')
    const body = new FormData()
    body.append('student_card', cardFile)
    try { await request('/student-card', { method: 'POST', body }); await refresh(); setCardFile(null); notify('学生证已提交，等待人工审核') }
    catch (err) { setError(err.message) } finally { setBusy(false) }
  }

  return (
    <Reveal className="onboarding">
      <div className="onboarding-title"><div><span className="section-kicker">GET STARTED</span><h2>完成这几步，开启你的相遇</h2></div><span>{completed} / 4</span></div>
      {error && <div className="form-error" role="alert">{error}</div>}
      <div className="onboarding-grid">
        <div className="onboard-item complete"><span className="round-icon"><Check size={20} /></span><strong>校园邮箱验证</strong><small>已完成</small></div>
        <div className={`onboard-item ${user.paid ? 'complete' : ''}`}>
          <span className="round-icon">¥</span><strong>9.9 元会费</strong><small>{user.paid ? '已完成' : '一次性会费'}</small>
          {!user.paid && <button onClick={onPayment}>查看会费方式 <ArrowRight size={14} /></button>}
        </div>
        <div className={`onboard-item ${user.card_status === 'approved' ? 'complete' : ''}`}>
          <span className="round-icon"><ShieldCheck size={20} /></span><strong>学生证审核</strong>
          <small>{user.card_status === 'pending' ? '审核中' : user.card_status === 'approved' ? '已通过' : user.card_status === 'rejected' ? '未通过，请重新提交' : '尚未提交'}</small>
          {user.card_status !== 'approved' && <div className="upload-control">
            <label><UploadCloud size={14} /> 选择图片或 PDF<input type="file" accept="image/jpeg,image/png,image/webp,application/pdf" onChange={event => setCardFile(event.target.files?.[0] || null)} /></label>
            {cardFile && <button onClick={uploadCard} disabled={busy}>提交审核</button>}
          </div>}
        </div>
        <div className={`onboard-item ${profileDone ? 'complete' : ''}`}>
          <span className="round-icon"><MessageCircle size={20} /></span><strong>填写个人介绍</strong><small>{profileDone ? '已完成' : user.gender && user.seeking ? '完善介绍与兴趣' : '补全交友方向与兴趣'}</small>
          {!profileDone && <button onClick={onProfile}>去填写 <ArrowRight size={14} /></button>}
        </div>
      </div>
      <p className="onboard-tip">兑换码可直接抵扣全部会费；使用收款码付款后，需提交交易单号并等待管理员核对到账。</p>
    </Reveal>
  )
}
