import { useCallback, useEffect, useState } from 'react'
import SiteHeader from './components/SiteHeader'
import SiteFooter from './components/SiteFooter'
import Toast from './components/Toast'
import AuthModal from './features/auth/AuthModal'
import LandingPage from './features/landing/LandingPage'
import Dashboard from './features/dashboard/Dashboard'
import { request } from './lib/api'
import useAutoRefresh from './hooks/useAutoRefresh'
import { preferredScrollBehavior } from './lib/motion'

export default function App() {
  const [user, setUser] = useState(null)
  const [modal, setModal] = useState('')
  const [notice, setNotice] = useState('')
  const [settingsRequest, setSettingsRequest] = useState(0)
  const [navigationRequest, setNavigationRequest] = useState({ id: 0, target: '' })

  useEffect(() => { request('/me').then(data => setUser(data.user)).catch(() => {}) }, [])

  const refresh = useCallback(async () => {
    const data = await request('/me')
    setUser(data.user)
    return data.user
  }, [])

  const backgroundRefresh = useCallback(async () => {
    try { await refresh() }
    catch (err) { if (err.status === 401) setUser(null) }
  }, [refresh])

  useAutoRefresh(backgroundRefresh, Boolean(user))

  function onAuth(nextUser) {
    setUser(nextUser)
    setSettingsRequest(0)
    setNavigationRequest({ id: 0, target: '' })
    setModal('')
    window.scrollTo({ top: 0, behavior: preferredScrollBehavior() })
  }

  async function logout() {
    try { await request('/logout', { method: 'POST' }); setUser(null); setSettingsRequest(0); setNavigationRequest({ id: 0, target: '' }); setNotice('已退出登录') }
    catch (err) { setNotice(err.message) }
  }

  return <>
    <SiteHeader user={user} onLogin={() => setModal('login')} onRegister={() => setModal('register')} onLogout={logout} onSettings={() => setSettingsRequest(value => value + 1)} onNotificationTarget={target => setNavigationRequest(current => ({ id: current.id + 1, target }))} />
    {user
      ? <Dashboard user={user} refresh={refresh} notify={setNotice} onLogout={logout} settingsRequest={settingsRequest} navigationRequest={navigationRequest} />
      : <LandingPage onJoin={() => setModal('register')} />}
    <SiteFooter />
    {modal && <AuthModal mode={modal} setMode={setModal} onClose={() => setModal('')} onAuth={onAuth} />}
    <Toast message={notice} onClose={() => setNotice('')} />
  </>
}
