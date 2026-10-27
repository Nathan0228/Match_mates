import { ArrowUpRight, Check, Sparkles } from 'lucide-react'
import Reveal from '../../components/Reveal'

const profiles = [
  { name: '晚风同学', major: '计算机科学 · 研一', tags: ['徒步', '电影', '咖啡'], note: '想找一个人一起走走雁栖湖的傍晚，聊聊代码以外的生活。', color: 'peach', score: '82' },
  { name: '橙子同学', major: '生命科学 · 研一', tags: ['摄影', '阅读', '旅行'], note: '喜欢记录路上的小事，也想认识愿意分享日常的人。', color: 'lavender', score: '76' },
  { name: '北岸同学', major: '物理学 · 研一', tags: ['羽毛球', '音乐', '做饭'], note: '周末喜欢运动和做饭，希望遇见聊得来、相处舒服的你。', color: 'mint', score: '68' },
]

export default function MatchPreview({ onJoin }) {
  return (
    <section className="match-section">
      <div className="wrap match-layout">
        <Reveal className="match-copy">
          <span className="section-kicker">MADE FOR YOU</span>
          <h2>兴趣相投，<br /><em>心意也相近。</em></h2>
          <p>填写简短介绍、兴趣与相处偏好后，见山会在你选择的交友方向中推荐同学。资料契合只是认识的起点，真正的故事由你们书写。</p>
          {['按彼此的交友方向匹配', '共同兴趣与相处偏好参与排序', '匹配分是认识的线索，不是好感概率'].map(text => <div className="feature-row" key={text}><span className="feature-icon"><Check size={17} /></span><span>{text}</span></div>)}
          <button className="under-btn" onClick={onJoin}>填写我的介绍 <ArrowUpRight size={16} /></button>
        </Reveal>
        <Reveal className="profile-stack" delay={90}>
          <div className="stack-label">一张未来可能出现的相遇卡片 <span>示意展示</span></div>
          {profiles.map((profile, index) => <div className={`example-card example-${index} ${profile.color}`} key={profile.name}>
            <div className="avatar"><span>{profile.name.slice(0, 1)}</span></div>
            <div className="example-detail">
              <div className="example-title"><h3>{profile.name}</h3><span className="score"><Sparkles size={12} />资料契合 {profile.score} 分</span></div>
              <div className="example-major">{profile.major}</div><p>{profile.note}</p>
              <div className="tag-row">{profile.tags.map(tag => <span key={tag}>{tag}</span>)}</div>
            </div>
          </div>)}
        </Reveal>
      </div>
    </section>
  )
}
