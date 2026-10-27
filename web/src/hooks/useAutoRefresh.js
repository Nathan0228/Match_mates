import { useEffect, useRef } from 'react'

export default function useAutoRefresh(refresh, enabled = true, intervalMs = 30000) {
  const running = useRef(false)

  useEffect(() => {
    if (!enabled) return
    let mounted = true
    const run = async () => {
      if (!mounted || document.visibilityState === 'hidden' || running.current) return
      running.current = true
      try { await refresh() } finally { running.current = false }
    }
    const onVisible = () => { if (document.visibilityState === 'visible') void run() }
    void run()
    const timer = window.setInterval(run, intervalMs)
    window.addEventListener('focus', run)
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      mounted = false
      window.clearInterval(timer)
      window.removeEventListener('focus', run)
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [refresh, enabled, intervalMs])
}
