import { useRef, useState } from 'react'
import { tmdbPosterUrl } from '../data/media'

const palettes = [
  ['#294e43', '#6c9a78', '#f7f6e8'],
  ['#715a4e', '#c88f74', '#fff5e7'],
  ['#4d5f70', '#9aafbb', '#f6f8f4'],
  ['#72675a', '#b8ae8e', '#fff9e9'],
  ['#6d5268', '#b68ca8', '#fff5f3'],
]

function toneFor(title) {
  let hash = 0
  for (const character of title) hash = (hash * 31 + character.charCodeAt(0)) | 0
  return palettes[Math.abs(hash) % palettes.length]
}

export default function MediaShelf({ category, items }) {
  const [active, setActive] = useState(0)
  const buttons = useRef([])
  if (!items?.length) return null
  const selected = Math.min(active, items.length - 1)

  function handleKeyDown(event, index) {
    const next = event.key === 'ArrowRight' ? (index + 1) % items.length
      : event.key === 'ArrowLeft' ? (index - 1 + items.length) % items.length
        : event.key === 'Home' ? 0 : event.key === 'End' ? items.length - 1 : null
    if (next === null) return
    event.preventDefault()
    setActive(next)
    buttons.current[next]?.focus()
  }

  return <div className={`media-shelf media-shelf--${category.kind}`}>
    <div className="media-shelf-track" role="group" aria-label={category.label}>
      {items.map((item, index) => {
        const [start, end, ink] = toneFor(item.title)
        const poster = ['movies', 'shows'].includes(category.key) ? tmdbPosterUrl(item.poster_path) : ''
        return <button
          type="button"
          key={`${item.title}-${index}`}
          ref={element => { buttons.current[index] = element }}
          className={`media-cover ${index === selected ? 'is-selected' : ''}${poster ? ' media-cover--with-poster' : ''}`}
          style={{ '--media-start': start, '--media-end': end, '--media-ink': ink, '--media-index': index }}
          aria-label={`${item.title}${item.creator ? `，${item.creator}` : ''}`}
          aria-pressed={index === selected}
          onClick={() => setActive(index)}
          onKeyDown={event => handleKeyDown(event, index)}
        >
          <span className="media-cover-surface" aria-hidden="true">
            {category.kind === 'book' && <span className="media-cover-spine">{item.title}</span>}
            <span className="media-cover-shape" />
            {poster && <img className="media-cover-poster" src={poster} alt="" loading="lazy" decoding="async" referrerPolicy="no-referrer" onError={event => { event.currentTarget.hidden = true }} onLoad={event => { event.currentTarget.hidden = false }} />}
            <span className="media-cover-type"><small>{category.mark}</small><strong>{item.title}</strong><em>{item.creator}</em></span>
          </span>
        </button>
      })}
    </div>
    <span className="media-shelf-ledge" aria-hidden="true" />
    <p className="media-shelf-caption"><span>{String(selected + 1).padStart(2, '0')} / {String(items.length).padStart(2, '0')}</span><strong>{items[selected].title}</strong>{items[selected].creator && <small>{items[selected].creator}</small>}</p>
  </div>
}
