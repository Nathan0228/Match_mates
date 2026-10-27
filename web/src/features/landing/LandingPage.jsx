import Hero from './Hero'
import HowItWorks from './HowItWorks'
import MatchPreview from './MatchPreview'
import Safety from './Safety'
import CallToAction from './CallToAction'

export default function LandingPage({ onJoin }) {
  return <main>
    <Hero onJoin={onJoin} onExplore={() => document.getElementById('how')?.scrollIntoView({ behavior: 'smooth' })} />
    <HowItWorks />
    <MatchPreview onJoin={onJoin} />
    <Safety />
    <CallToAction onJoin={onJoin} />
  </main>
}
