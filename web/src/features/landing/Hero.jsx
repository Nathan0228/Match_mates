import { ArrowRight, ArrowUpRight, ChevronDown } from 'lucide-react'
import Reveal from '../../components/Reveal'

export default function Hero({ onJoin, onExplore }) {
  return (
    <section className="hero" id="about">
      <div className="wrap hero-inner">
        <Reveal className="hero-copy">
          <div className="eyebrow"><span className="eyebrow-line" /> 国科大 · 一段新的相遇</div>
          <h1>山水有相逢，<br /><em>我们</em>刚好遇见。</h1>
          <p className="hero-sub">研一在雁栖湖的一年，除了学习与成长，也值得有人陪你走过傍晚的湖边。这里是属于国科大学生的认真交友空间。</p>
          <div className="hero-buttons">
            <button className="pill-btn" onClick={onJoin}>开启一段相遇 <ArrowRight size={18} /></button>
            <button className="quiet-btn" onClick={onExplore}>先了解一下 <ArrowUpRight size={17} /></button>
          </div>
          <div className="hero-trust"><span className="trust-avatars"><i>山</i><i>湖</i><i>你</i></span><span>校园邮箱认证 · 人工审核 · 双向同意</span></div>
        </Reveal>
        <Reveal className="hero-art" delay={100} role="img" aria-label="雁栖湖风景插画">
          <div className="sun" /><div className="cloud cloud-one" /><div className="cloud cloud-two" />
          <div className="mountain mountain-back" /><div className="mountain mountain-front" />
          <div className="lake" /><div className="lake-line l1" /><div className="lake-line l2" /><div className="lake-line l3" />
          <div className="shore" />
          <div className="person person-a"><span className="head" /><span className="body" /></div>
          <div className="person person-b"><span className="head" /><span className="body" /></div>
          <div className="sparkle s1">✳</div><div className="sparkle s2">✳</div>
          <div className="art-caption">YANQI LAKE · BEIJING</div>
        </Reveal>
      </div>
      <div className="hero-bottom wrap"><span>为每一种心动，留一处安心的空间</span><span className="scroll-cue">向下探索 <ChevronDown size={15} /></span></div>
    </section>
  )
}
