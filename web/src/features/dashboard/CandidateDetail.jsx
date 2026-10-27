import { ArrowLeft, Heart, Mail, MapPin, Sparkles } from 'lucide-react'
import Avatar from '../../components/Avatar'
import EmptyState from '../../components/EmptyState'
import Reveal from '../../components/Reveal'
import MediaShelf from '../../components/MediaShelf'
import { mediaCategories } from '../../data/media'
import { contacts, paces, rhythms } from '../../data/preferences'

function optionLabel(options, value) {
  return options.find(option => option.value === value)?.label || ''
}

function DetailSection({ title, children }) {
  return <Reveal as="section" className="candidate-detail-section"><h3>{title}</h3>{children}</Reveal>
}

export default function CandidateDetail({ person, match, onBack, onLike, liking }) {
  if (!person) return <EmptyState title="这份资料暂时无法查看" text="对方的资料或审核状态可能已更新，请返回发现页看看。" onClick={onBack} button="返回发现" />

  const preferences = person.preferences || {}
  const preferenceItems = [
    ['认识的节奏', optionLabel(paces, preferences.pace)],
    ['联系的习惯', optionLabel(contacts, preferences.contact)],
    ['社交充电方式', preferences.energy ? `${preferences.energy} / 5 · ${preferences.energy <= 2 ? '偏向独处' : preferences.energy >= 4 ? '偏向社交' : '两者都喜欢'}` : ''],
    ['平常作息', optionLabel(rhythms, preferences.rhythm)],
  ].filter(([, value]) => value)

  return <article className="candidate-detail">
    <button className="detail-back" type="button" onClick={onBack}><ArrowLeft size={17} /> 返回发现</button>
    <div className="candidate-detail-hero">
      <Avatar person={person} className="detail-avatar" />
      <div className="detail-heading">
        <span className="section-kicker">MEET SOMEONE NEW</span>
        <h2>{person.nickname}</h2>
        <p><MapPin size={15} /> {person.major || '国科大学生'} · {person.campus || '雁栖湖'}{preferences.mbti ? ` · ${preferences.mbti}` : ''}</p>
        <span className="detail-score"><Sparkles size={16} /> 资料契合 {Math.round(person.score * 100)} 分</span>
      </div>
    </div>

    {person.reasons?.length > 0 && <div className="detail-reasons"><strong>你们的共同点</strong><div className="match-reasons">{person.reasons.map(reason => <span key={reason}>{reason}</span>)}</div></div>}

    <DetailSection title="关于 TA"><p className="detail-text">{person.intro}</p></DetailSection>
    {preferences.daily && <DetailSection title="一个普通周末"><p className="detail-text">{preferences.daily}</p></DetailSection>}
    {preferences.together && <DetailSection title="期待一起做"><p className="detail-text">{preferences.together}</p></DetailSection>}
    {preferences.values?.length > 0 && <DetailSection title="看重的相处方式"><div className="detail-tags">{preferences.values.map(value => <span key={value}>{value}</span>)}</div></DetailSection>}
    <DetailSection title="喜欢的事"><div className="detail-tags">{(person.hobbies || []).map(hobby => <span key={hobby}>{hobby}</span>)}</div></DetailSection>
    {mediaCategories.filter(category => person.media?.[category.key]?.length).map(category => <DetailSection key={category.key} title={category.label}><MediaShelf category={category} items={person.media[category.key]} /></DetailSection>)}
    {preferenceItems.length > 0 && <DetailSection title="相处偏好"><dl className="detail-preferences">{preferenceItems.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl></DetailSection>}

    <div className="detail-action">
      <p>资料契合分只反映填写的共同点。认识以后，再由你们自己判断是否合拍。</p>
      {match ? <a className="detail-email" href={`mailto:${match.email}`}><Mail size={18} /> 双向心动，发邮件给 TA</a>
        : <button className="pill-btn" type="button" disabled={person.liked || liking} onClick={() => onLike(person.id)}><Heart size={18} fill={person.liked ? 'currentColor' : 'none'} />{liking ? '发送中…' : person.liked ? '已表达想认识，等待回应' : '想认识 TA'}</button>}
    </div>
  </article>
}
