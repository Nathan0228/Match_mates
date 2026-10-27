import { lazy, Suspense, useCallback, useEffect, useRef, useState } from 'react'
import { CreditCard, Heart, PencilLine, ShieldCheck, Sparkles } from 'lucide-react'
import { request } from '../../lib/api'
import { preferredScrollBehavior } from '../../lib/motion'
import useAutoRefresh from '../../hooks/useAutoRefresh'
import Onboarding from './Onboarding'
import Discovery from './Discovery'

const CandidateDetail = lazy(() => import('./CandidateDetail'))
const Matches = lazy(() => import('./Matches'))
const ProfileEditor = lazy(() => import('./ProfileEditor'))
const AdminReviews = lazy(() => import('./AdminReviews'))
const PaymentCenter = lazy(() => import('./payment/PaymentCenter'))
const SettingsPage = lazy(() => import('./SettingsPage'))

function NavIcon({ icon: Icon, active, animationId }) {
  return <span className="dash-nav-icon" aria-hidden="true">
    <Icon className="dash-nav-icon-outline" size={19} />
    <Icon className="dash-nav-icon-fill" size={19} fill="currentColor" key={active ? animationId : 'inactive'} />
  </span>
}

export default function Dashboard({ user, refresh, notify, onLogout, settingsRequest = 0, navigationRequest = { id: 0, target: '' } }) {
  const [tab, setTab] = useState(() => user.is_admin ? 'admin' : 'discover')
  const [selectedCandidateId, setSelectedCandidateId] = useState(null)
  const [likingId, setLikingId] = useState(null)
  const [candidates, setCandidates] = useState([])
  const [matches, setMatches] = useState([])
  const [error, setError] = useState('')
  const [navAnimationId, setNavAnimationId] = useState(0)
  const lastSettingsRequest = useRef(settingsRequest)
  const lastNavigationRequest = useRef(navigationRequest.id)
  const returnTab = useRef(user.is_admin ? 'admin' : 'discover')
  const [adminViewRequest, setAdminViewRequest] = useState({ id: 0, view: 'queue' })
  const active = user.status === 'active'

  useEffect(() => {
    if (lastSettingsRequest.current === settingsRequest) return
    lastSettingsRequest.current = settingsRequest
    setSelectedCandidateId(null)
    setTab(current => {
      if (current !== 'settings') returnTab.current = current
      return 'settings'
    })
    window.requestAnimationFrame(() => document.querySelector('.dash-content')?.scrollIntoView({ behavior: preferredScrollBehavior(), block: 'start' }))
  }, [settingsRequest])

  useEffect(() => {
    if (lastNavigationRequest.current === navigationRequest.id) return
    lastNavigationRequest.current = navigationRequest.id
    const target = navigationRequest.target
    setSelectedCandidateId(null)
    if (user.is_admin && (target === 'admin:queue' || target === 'admin:payments')) {
      setAdminViewRequest({ id: navigationRequest.id, view: target.slice(6) })
      setTab('admin')
    } else if (['discover', 'matches', 'profile', 'payment'].includes(target)) {
      setTab(target)
    }
    window.requestAnimationFrame(() => document.querySelector('.dash-content')?.scrollIntoView({ behavior: preferredScrollBehavior(), block: 'start' }))
  }, [navigationRequest, user.is_admin])

  const refreshSocial = useCallback(async () => {
    try {
      const [candidateData, matchData] = await Promise.all([request('/candidates'), request('/matches')])
      setCandidates(candidateData.candidates || [])
      setMatches(matchData.matches || [])
      setError('')
    } catch (err) { setError(err.message) }
  }, [])

  useAutoRefresh(refreshSocial, active)

  async function refreshAfterUpdate() {
    const nextUser = await refresh()
    if (nextUser.status === 'active') await refreshSocial()
    return nextUser
  }

  async function like(id) {
    setError('')
    setLikingId(id)
    try {
      const result = await request(`/likes/${id}`, { method: 'POST' })
      setCandidates(items => items.map(person => person.id === id ? { ...person, liked: true } : person))
      if (result.mutual) {
        notify('你们互相表达了想认识！邮箱已解锁')
        const data = await request('/matches')
        setMatches(data.matches || [])
      } else notify('心意已送达，等待对方回应')
    } catch (err) { setError(err.message) } finally { setLikingId(null) }
  }

  function openTab(nextTab) {
    setSelectedCandidateId(null)
    setTab(nextTab)
    setNavAnimationId(id => id + 1)
    if (window.matchMedia('(max-width: 700px)').matches) {
      window.requestAnimationFrame(() => document.querySelector('.dash-content')?.scrollIntoView({
        behavior: preferredScrollBehavior(),
        block: 'start',
      }))
    }
  }

  function viewCandidate(id) {
    setSelectedCandidateId(id)
    window.requestAnimationFrame(() => document.querySelector('.dash-content')?.scrollIntoView({ behavior: preferredScrollBehavior(), block: 'start' }))
  }

  return (
    <main className="dashboard" id="dashboard">
      <div className="wrap">
        {!active && <Onboarding user={user} refresh={refreshAfterUpdate} notify={notify} onProfile={() => openTab('profile')} onPayment={() => openTab('payment')} />}
        <div className="dashboard-grid">
          <nav className="dash-side" aria-label="主要功能" style={{ '--tab-count': user.is_admin ? 5 : 4 }}>
            <button type="button" aria-label="发现同频" className={tab === 'discover' ? 'selected' : ''} aria-current={tab === 'discover' ? 'page' : undefined} onClick={() => openTab('discover')}><NavIcon icon={Sparkles} active={tab === 'discover'} animationId={navAnimationId} /><span className="nav-label-full">发现同频</span><span className="nav-label-mobile"></span></button>
            <button type="button" aria-label="双向心动" className={tab === 'matches' ? 'selected' : ''} aria-current={tab === 'matches' ? 'page' : undefined} onClick={() => openTab('matches')}><NavIcon icon={Heart} active={tab === 'matches'} animationId={navAnimationId} /><span className="nav-label-full">双向心动</span><span className="nav-label-mobile"></span></button>
            <button type="button" aria-label="我的介绍" className={tab === 'profile' || tab === 'settings' ? 'selected' : ''} aria-current={tab === 'profile' || tab === 'settings' ? 'page' : undefined} onClick={() => openTab('profile')}><NavIcon icon={PencilLine} active={tab === 'profile' || tab === 'settings'} animationId={navAnimationId} /><span className="nav-label-full">我的介绍</span><span className="nav-label-mobile"></span></button>
            <button type="button" aria-label={user.paid ? '会费记录' : '办理会费'} className={tab === 'payment' ? 'selected' : ''} aria-current={tab === 'payment' ? 'page' : undefined} onClick={() => openTab('payment')}><NavIcon icon={CreditCard} active={tab === 'payment'} animationId={navAnimationId} /><span className="nav-label-full">{user.paid ? '会费记录' : '办理会费'}</span><span className="nav-label-mobile"></span></button>
            {user.is_admin && <button type="button" aria-label="管理中心" className={tab === 'admin' ? 'selected' : ''} aria-current={tab === 'admin' ? 'page' : undefined} onClick={() => openTab('admin')}><NavIcon icon={ShieldCheck} active={tab === 'admin'} animationId={navAnimationId} /><span className="nav-label-full">管理中心</span><span className="nav-label-mobile"></span></button>}
          </nav>
          <div className="dash-content">
            {error && <div className="form-error" role="alert">{error}</div>}
            <Suspense fallback={<div className="dash-loading" role="status">内容加载中…</div>}>
            {tab === 'discover' && (selectedCandidateId && active
              ? <CandidateDetail person={candidates.find(person => person.id === selectedCandidateId)} match={matches.find(person => person.id === selectedCandidateId)} onBack={() => setSelectedCandidateId(null)} onLike={like} liking={likingId === selectedCandidateId} />
              : <Discovery active={active} matchingEnabled={user.matching_enabled !== false} candidates={candidates} onView={viewCandidate} onProfile={() => openTab('profile')} onSettings={() => openTab('settings')} />)}
            {tab === 'matches' && <Matches matches={matches} onDiscover={() => openTab('discover')} />}
            {tab === 'profile' && <ProfileEditor user={user} refresh={refreshAfterUpdate} notify={notify} />}
            {tab === 'payment' && <PaymentCenter user={user} refresh={refreshAfterUpdate} notify={notify} />}
            {tab === 'admin' && user.is_admin && <AdminReviews notify={notify} viewRequest={adminViewRequest} />}
            {tab === 'settings' && <SettingsPage user={user} refresh={refreshAfterUpdate} notify={notify} onBack={() => openTab(returnTab.current)} onProfile={() => openTab('profile')} onPayment={() => openTab('payment')} onLogout={onLogout} />}
            </Suspense>
          </div>
        </div>
      </div>
    </main>
  )
}
