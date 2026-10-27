import { ArrowUpRight } from 'lucide-react'
import Reveal from '../../components/Reveal'

export default function CallToAction({ onJoin }) {
  return (
    <section className="cta-section">
      <Reveal className="wrap cta-inner">
        <span className="section-kicker">YOUR STORY STARTS HERE</span>
        <h2>下次路过雁栖湖，<br />也许有人<em>刚好同行。</em></h2>
        <p>从一封校园邮箱开始，认真认识一个人。</p>
        <button className="pill-btn light" onClick={onJoin}>加入见山 <ArrowUpRight size={18} /></button>
        <div className="cta-doodle">♡</div>
      </Reveal>
    </section>
  )
}
