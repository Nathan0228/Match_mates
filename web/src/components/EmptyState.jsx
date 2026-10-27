import { ArrowUpRight } from 'lucide-react'

export default function EmptyState({ title, text, onClick, button }) {
  return (
    <div className="empty-state">
      <span className="empty-illustration">♡</span>
      <h3>{title}</h3>
      <p>{text}</p>
      {button && <button className="under-btn" onClick={onClick}>{button} <ArrowUpRight size={16} /></button>}
    </div>
  )
}
