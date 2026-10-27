import { useEffect, useRef } from 'react'
import { X } from 'lucide-react'

export default function Toast({ message, onClose }) {
  const closeRef = useRef(onClose)
  closeRef.current = onClose

  useEffect(() => {
    if (!message) return
    const timer = window.setTimeout(() => closeRef.current(), 3000)
    return () => window.clearTimeout(timer)
  }, [message])

  if (!message) return null
  return <div className="toast" key={message} role="status">{message}<button onClick={onClose} aria-label="关闭"><X size={14} /></button></div>
}
