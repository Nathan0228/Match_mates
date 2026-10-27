import { useEffect, useRef, useState } from 'react'

export default function Reveal({ as: Element = 'div', className = '', delay = 0, children, style, ...props }) {
  const element = useRef(null)
  const [visible, setVisible] = useState(() => typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches)

  useEffect(() => {
    if (visible) return
    if (!('IntersectionObserver' in window)) { setVisible(true); return }
    const observer = new IntersectionObserver(entries => {
      if (entries.some(entry => entry.isIntersecting)) {
        setVisible(true)
        observer.disconnect()
      }
    }, { threshold: 0.08, rootMargin: '0px 0px -24px 0px' })
    if (element.current) observer.observe(element.current)
    return () => observer.disconnect()
  }, [visible])

  return <Element ref={element} className={`scroll-reveal ${visible ? 'is-visible' : ''} ${className}`.trim()} style={{ ...style, '--reveal-delay': `${delay}ms` }} {...props}>{children}</Element>
}
