import { Heart, Mail, Sparkles } from 'lucide-react'
import Reveal from '../../components/Reveal'

const steps = [
  { number: '01', title: '用校园邮箱来敲门', text: '仅接受 @mails.ucas.ac.cn 邮箱，让每一次相遇都从真实的校园身份开始。', icon: Mail, doodle: '✉' },
  { number: '02', title: '认真介绍自己', text: '写下你的日常、兴趣与期待。完成会费和学生证审核后，才会出现在匹配中。', icon: Sparkles, doodle: '✺' },
  { number: '03', title: '从双向心动开始', text: '只展示与你契合的人。双方都表达想认识，才交换校园邮箱。', icon: Heart, doodle: '♡' },
]

export default function HowItWorks() {
  return (
    <section className="intro-section" id="how">
      <div className="wrap">
        <Reveal className="section-top">
          <div><span className="section-kicker">HOW IT WORKS</span><h2>从认识自己，<br />到遇见<em>合适的人。</em></h2></div>
          <p>我们相信，好的关系不需要仓促开始。留一点时间说说自己，让相遇更有方向。</p>
        </Reveal>
        <div className="step-grid">
          {steps.map((Step, index) => <Reveal className="step-card" delay={index * 75} key={Step.number}>
            <div className="step-head"><span>{Step.number}</span><Step.icon size={22} strokeWidth={1.6} /></div>
            <div className="step-doodle"><span className={`doodle doodle-${Step.number}`}>{Step.doodle}</span></div>
            <h3>{Step.title}</h3><p>{Step.text}</p>
          </Reveal>)}
        </div>
      </div>
    </section>
  )
}
