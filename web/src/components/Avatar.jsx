import { avatarSrc } from '../lib/api'

export default function Avatar({ person, className = 'candidate-avatar' }) {
  const src = avatarSrc(person)
  return (
    <span className={className} aria-label={`${person.nickname || '同学'}的头像`}>
      {src ? <img src={src} alt="" loading="lazy" decoding="async" /> : (person.nickname || '同').slice(0, 1)}
    </span>
  )
}
