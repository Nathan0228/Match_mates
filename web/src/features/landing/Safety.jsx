import { BadgeCheck, Heart, LockKeyhole, ShieldCheck } from 'lucide-react'
import Reveal from '../../components/Reveal'

const safeguards = [
  { icon: BadgeCheck, title: '真实校园身份', text: '国科大邮箱验证、会费与学生证人工审核' },
  { icon: LockKeyhole, title: '联系方式由你决定', text: '仅当双方互相表达意愿，邮箱才会交换' },
  { icon: ShieldCheck, title: '不同交友方向彼此隔离', text: '只在双方选择互相符合的范围内出现' },
]

export default function Safety() {
  return (
    <section className="safety-section" id="safety">
      <div className="wrap safety-layout">
        <Reveal className="safety-art">
          <div className="shield-ring ring-outer" /><div className="shield-ring ring-inner" />
          <div className="safety-heart"><Heart size={68} strokeWidth={1.15} /></div>
          <span className="orbit orbit-a">真实身份</span><span className="orbit orbit-b">自主选择</span><span className="orbit orbit-c">隐私保护</span>
        </Reveal>
        <Reveal className="safety-copy" delay={80}>
          <span className="section-kicker">A SAFER SPACE</span>
          <h2>让靠近，<br />从<em>安心</em>开始。</h2>
          <p>认真交友，需要一点信任，也需要边界感。我们让每一步都清楚、可控。</p>
          <div className="safety-list">{safeguards.map(Item => <div key={Item.title}><Item.icon size={22} /><span><strong>{Item.title}</strong><small>{Item.text}</small></span></div>)}</div>
        </Reveal>
      </div>
    </section>
  )
}
