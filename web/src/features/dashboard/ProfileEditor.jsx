import { useEffect, useState } from 'react'
import { ArrowRight, Camera } from 'lucide-react'
import Avatar from '../../components/Avatar'
import ChoiceGroup from '../../components/ChoiceGroup'
import { genderOptions, seekingOptions } from '../../data/identity'
import { interests } from '../../data/interests'
import { mediaCategories, normalizeMedia } from '../../data/media'
import { request } from '../../lib/api'
import MatchPreferencesEditor from './MatchPreferencesEditor'
import MediaCollectionsEditor from './MediaCollectionsEditor'

export default function ProfileEditor({ user, refresh, notify }) {
  const [form, setForm] = useState(() => profileForm(user))
  const [dirty, setDirty] = useState(false)
  const [avatarFile, setAvatarFile] = useState(null)
  const [preview, setPreview] = useState('')
  const [busy, setBusy] = useState(false)
  const [avatarBusy, setAvatarBusy] = useState(false)
  const [error, setError] = useState('')
  useEffect(() => {
    if (!avatarFile) { setPreview(''); return }
    const url = URL.createObjectURL(avatarFile)
    setPreview(url)
    return () => URL.revokeObjectURL(url)
  }, [avatarFile])
  useEffect(() => { if (!dirty) setForm(profileForm(user)) }, [user, dirty])
  const set = (key, value) => { setDirty(true); setForm(current => ({ ...current, [key]: value })) }
  const toggle = tag => { setDirty(true); setForm(current => ({ ...current, hobbies: current.hobbies.includes(tag) ? current.hobbies.filter(item => item !== tag) : [...current.hobbies, tag] })) }
  const setPreferences = preferences => { setDirty(true); setForm(current => ({ ...current, preferences })) }
  const setMedia = update => { setDirty(true); setForm(current => ({ ...current, media: typeof update === 'function' ? update(current.media) : update })) }

  function pickAvatar(event) {
    const file = event.target.files?.[0]
    if (!file) return
    if (!['image/jpeg', 'image/png'].includes(file.type)) { setError('头像仅支持 JPG 或 PNG'); return }
    if (file.size > 2 * 1024 * 1024) { setError('头像文件不得超过 2 MB'); return }
    setError('')
    setAvatarFile(file)
  }

  async function uploadAvatar() {
    if (!avatarFile) return
    setError('')
    setAvatarBusy(true)
    const body = new FormData()
    body.append('avatar', avatarFile)
    try {
      await request('/avatar', { method: 'POST', body })
      setAvatarFile(null)
      await refresh()
      notify('头像已更新')
    } catch (err) { setError(err.message) } finally { setAvatarBusy(false) }
  }

  async function save(event) {
    event.preventDefault()
    setError('')
    const media = normalizeMedia(form.media)
    for (const { key, label } of mediaCategories) {
      if (media[key].some(item => !item.title.trim() && item.creator.trim())) {
        setError(`请为“${label}”填写作品名称，或移除空白条目`)
        return
      }
      media[key] = media[key].filter(item => item.title.trim()).map(item => ({ ...item, title: item.title.trim(), creator: item.creator.trim() }))
    }
    setBusy(true)
    try {
      await request('/profile', { method: 'PUT', body: JSON.stringify({ ...form, media }) })
      await refresh()
      setDirty(false)
      notify('个人资料已保存')
    } catch (err) { setError(err.message) } finally { setBusy(false) }
  }

  return <>
    <h2>我的介绍</h2><p className="dash-description">写得越具体，越容易遇见聊得来的人。</p>
    <form className="profile-form" onSubmit={save}>
      <div className="avatar-editor">
        {preview ? <span className="profile-avatar"><img src={preview} alt="头像预览" /></span> : <Avatar person={user} className="profile-avatar" />}
        <div><strong>我的头像</strong><small>JPG 或 PNG，最多 2 MB。</small>
          <label className="avatar-pick"><Camera size={15} /> 选择照片<input type="file" accept="image/jpeg,image/png" onChange={pickAvatar} /></label>
          {avatarFile && <small className="avatar-filename">已选择：{avatarFile.name}</small>}
          {avatarFile && <button className="avatar-upload" type="button" onClick={uploadAvatar} disabled={avatarBusy}>{avatarBusy ? '上传中…' : '上传头像'}</button>}
        </div>
      </div>
      {error && <div className="form-error" role="alert">{error}</div>}
      <div className="field-pair">
        <label>昵称<input required maxLength="24" value={form.nickname} onChange={event => set('nickname', event.target.value)} /></label>
        <label>专业 / 研究方向<input maxLength="60" value={form.major} onChange={event => set('major', event.target.value)} placeholder="例如：计算机科学" /></label>
      </div>
      <div className="choice-grid choice-grid--identity">
        <ChoiceGroup label="我的性别" hint={user.gender ? '注册时已确定' : '请选择'} options={genderOptions} value={form.gender} onChange={value => set('gender', value)} disabled={Boolean(user.gender)} required />
        <ChoiceGroup label="希望认识" hint={user.seeking ? '注册时已确定' : '请选择'} options={seekingOptions} value={form.seeking} onChange={value => set('seeking', value)} disabled={Boolean(user.seeking)} required />
      </div>
      <p className="field-hint">只有交友方向互相符合的用户会彼此出现。性别和交友方向确定后不可修改。</p>
      <label>所在校区<input maxLength="40" value={form.campus} onChange={event => set('campus', event.target.value)} placeholder="雁栖湖" /></label>
      <label>先用几句话介绍自己 <small>建议 20–240 字，写最想让对方认识的你</small><textarea required minLength="20" maxLength={Math.max(240, user.intro?.length || 0)} value={form.intro} onChange={event => set('intro', event.target.value)} placeholder="我最近在忙什么、喜欢怎样的生活？不用写得很长。" rows="3" /></label>
      <label>兴趣爱好 <small>选择至少 2 项，也可以在自我介绍里写更多</small></label>
      <div className="interest-list">{interests.map(tag => <button type="button" className={form.hobbies.includes(tag) ? 'picked' : ''} key={tag} onClick={() => toggle(tag)}>{tag}</button>)}</div>
      <MediaCollectionsEditor media={form.media} onChange={setMedia} />
      <MatchPreferencesEditor preferences={form.preferences} onChange={setPreferences} />
      <button className="pill-btn save-btn" disabled={busy}>{busy ? '保存中…' : '保存介绍'} <ArrowRight size={17} /></button>
    </form>
  </>
}

function profileForm(user) {
  return { nickname: user.nickname || '', gender: user.gender || '', seeking: user.seeking || '', major: user.major || '', campus: user.campus || '雁栖湖', intro: user.intro || '', hobbies: user.hobbies || [], media: normalizeMedia(user.media), preferences: { values: [], pace: '', contact: '', energy: 0, rhythm: '', mbti: '', daily: '', together: '', ...user.preferences } }
}
