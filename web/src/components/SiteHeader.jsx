import { useEffect, useRef, useState } from 'react'
import { ArrowUpRight, LogOut, Menu, Settings } from 'lucide-react'
import Avatar from './Avatar'
import Brand from './Brand'
import NotificationMenu from './NotificationMenu'
import { preferredScrollBehavior } from '../lib/motion'

export default function SiteHeader({ user, onLogin, onRegister, onLogout, onSettings, onNotificationTarget }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const [activeMenu, setActiveMenu] = useState('')
  const actionsRef = useRef(null)
  useEffect(() => { if (user) setMenuOpen(false); else setActiveMenu('') }, [user])
  useEffect(() => {
    if (!activeMenu) return
    const dismissOutside = event => { if (!actionsRef.current?.contains(event.target)) setActiveMenu('') }
    const dismissEscape = event => { if (event.key === 'Escape') setActiveMenu('') }
    document.addEventListener('pointerdown', dismissOutside)
    document.addEventListener('keydown', dismissEscape)
    return () => {
      document.removeEventListener('pointerdown', dismissOutside)
      document.removeEventListener('keydown', dismissEscape)
    }
  }, [activeMenu])
  const jump = id => { document.getElementById(id)?.scrollIntoView({ behavior: preferredScrollBehavior() }); setMenuOpen(false) }
  const home = () => window.scrollTo({ top: 0, behavior: preferredScrollBehavior() })

  return (
    <header className="site-header">
      <div className="wrap header-inner">
        <Brand onClick={home} />
        {!user && <nav className={menuOpen ? 'nav open' : 'nav'}>
          <button onClick={() => jump('about')}>关于见山</button>
          <button onClick={() => jump('how')}>如何相遇</button>
          <button onClick={() => jump('safety')}>安心守则</button>
        </nav>}
        <div className="header-actions" ref={actionsRef}>
          {user ? <>
            <NotificationMenu open={activeMenu === 'notifications'} onToggle={() => setActiveMenu(current => current === 'notifications' ? '' : 'notifications')} onClose={() => setActiveMenu('')} onNavigate={onNotificationTarget} />
            <div className="account-menu">
            <button className="account-trigger" onClick={() => setActiveMenu(current => current === 'account' ? '' : 'account')} aria-label="账户菜单" aria-expanded={activeMenu === 'account'} aria-haspopup="true">
              <Avatar person={user} className="header-avatar" />
            </button>
            {activeMenu === 'account' && <div className="account-dropdown" aria-label="账户选项">
              <button type="button" onClick={() => { setActiveMenu(''); onSettings() }}><Settings size={15} aria-hidden="true" />设置</button>
              <button type="button" onClick={() => { setActiveMenu(''); onLogout() }}><LogOut size={15} aria-hidden="true" />退出登录</button>
            </div>}
          </div></> : <>
            <button className="text-btn" onClick={onLogin}>登录</button>
            <button className="pill-btn small" onClick={onRegister}>加入见山 <ArrowUpRight size={15} /></button>
          </>}
        </div>
        {!user && <button className="menu-btn" onClick={() => setMenuOpen(!menuOpen)} aria-label="打开菜单"><Menu size={23} /></button>}
      </div>
    </header>
  )
}
