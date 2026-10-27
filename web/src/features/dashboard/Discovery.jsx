import { ArrowUpRight, Sparkles } from 'lucide-react'
import Avatar from '../../components/Avatar'
import EmptyState from '../../components/EmptyState'
import Reveal from '../../components/Reveal'

export default function Discovery({ active, matchingEnabled = true, candidates, onView, onProfile, onSettings }) {
  return <>
    <h2>发现同频的人</h2>
    <p className="dash-description">在双方交友方向一致的同学中，按共同兴趣与相处偏好排序。资料契合分仅供参考。</p>
    {!active ? <EmptyState title="完成审核后，就可以开始发现同频的人" text="先填写个人资料，完成会费和学生证审核。" onClick={onProfile} button="完善资料" />
      : !matchingEnabled ? <EmptyState title="匹配已暂停" text="你的资料暂时不会出现在新的推荐中，也无法发出新的想认识。已有的双向心动仍可查看。" onClick={onSettings} button="前往设置" />
      : candidates.length ? <div className="candidate-grid">{candidates.map((person, index) => <Reveal className="candidate-card" delay={Math.min(index % 4, 3) * 55} key={person.id}>
        <div className="candidate-top"><Avatar person={person} /><span className="score"><Sparkles size={13} />资料契合 {Math.round(person.score * 100)} 分</span></div>
        <h3>{person.nickname}</h3><small>{person.major || '国科大学生'} · {person.campus || '雁栖湖'}{person.preferences?.mbti ? ` · ${person.preferences.mbti}` : ''}</small>
        <p className="candidate-excerpt">{person.intro || '这个人还在认真填写介绍。'}</p>
        {person.reasons?.length > 0 && <div className="match-reasons">{person.reasons.slice(0, 2).map(reason => <span key={reason}>{reason}</span>)}</div>}
        <div className="tag-row">{(person.hobbies || []).slice(0, 3).map(tag => <span key={tag}>{tag}</span>)}{person.hobbies?.length > 3 && <span>+{person.hobbies.length - 3}</span>}</div>
        <button className="candidate-view" type="button" onClick={() => onView(person.id)}>查看完整资料 <ArrowUpRight size={16} /></button>
      </Reveal>)}</div>
      : <EmptyState title="暂时没有符合双方交友方向的同学" text="可以完善资料，过一阵再来看看。" onClick={onProfile} button="完善我的介绍" />}
  </>
}
