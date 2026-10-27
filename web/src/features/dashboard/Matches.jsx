import { Mail } from 'lucide-react'
import Avatar from '../../components/Avatar'
import EmptyState from '../../components/EmptyState'
import Reveal from '../../components/Reveal'

export default function Matches({ matches, onDiscover }) {
  return <>
    <h2>双向心动</h2>
    <p className="dash-description">只有彼此都选择“想认识”，这里才会出现联系方式。</p>
    {matches.length ? <div className="matched-list">{matches.map((person, index) => <Reveal className="matched-person" delay={Math.min(index, 3) * 55} key={person.id}>
      <Avatar person={person} />
      <div><strong>{person.nickname}</strong><small>{person.major || '国科大学生'}</small></div>
      <a href={`mailto:${person.email}`}><Mail size={16} />{person.email}</a>
    </Reveal>)}</div> : <EmptyState title="心动还在路上" text="在发现页表达你的心意，等对方也选择你时，邮箱就会解锁。" onClick={onDiscover} button="去发现" />}
  </>
}
