import { useEffect, useState } from 'react'
import { Check, LoaderCircle } from 'lucide-react'
import { request } from '../../lib/api'
import { tmdbPosterUrl } from '../../data/media'

export default function TMDBLookup({ kind, title, selected, onSelect }) {
  const [results, setResults] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [searched, setSearched] = useState(false)

  useEffect(() => {
    const query = title.trim()
    if (query.length < 2) {
      setResults([])
      setLoading(false)
      setError('')
      setSearched(false)
      return
    }
    if (selected?.tmdb_id) return
    let active = true
    const controller = new AbortController()
    setResults([])
    setSearched(false)
    setError('')
    const timer = window.setTimeout(async () => {
      setLoading(true)
      try {
        const data = await request(`/media/search?kind=${kind}&q=${encodeURIComponent(query)}`, { signal: controller.signal })
        if (!active) return
        const matches = data.results || []
        setResults(matches)
        setSearched(true)
        const firstWithPoster = matches.find(match => match.poster_path)
        if (firstWithPoster && !selected?.tmdb_id) onSelect(firstWithPoster)
      } catch (err) {
        if (active && err.name !== 'AbortError') setError(err.message)
      } finally { if (active) setLoading(false) }
    }, 650)
    return () => { active = false; window.clearTimeout(timer); controller.abort() }
  }, [kind, title, selected?.tmdb_id])

  if (!title.trim()) return null
  const selectedPoster = tmdbPosterUrl(selected?.poster_path, 'w92')
  return <div className="media-lookup">
    {selected?.tmdb_id && <div className="media-lookup-selected">
      {selectedPoster && <img src={selectedPoster} alt="" loading="lazy" referrerPolicy="no-referrer" />}
      <span>{selectedPoster ? '已匹配 TMDB 海报' : '已找到 TMDB 作品，暂无海报'}</span>
      <button type="button" onClick={() => onSelect(null)}>重新匹配</button>
    </div>}
    {loading && <p className="media-lookup-status"><LoaderCircle size={13} className="media-lookup-spin" aria-hidden="true" />正在查找 TMDB 作品…</p>}
    {error && <p className="media-lookup-status media-lookup-error" role="status">{error}；仍可保存作品名称。</p>}
    {searched && !results.length && <p className="media-lookup-status">暂未找到作品；仍可保存作品名称。</p>}
    {results.length > 0 && <div className="media-lookup-results" aria-label="TMDB 匹配结果">
      <span>确认海报是否对应你的作品</span>
      <div>{results.map(result => <button type="button" key={result.id} className={selected?.tmdb_id === result.id ? 'is-selected' : ''} onClick={() => onSelect(result)} aria-pressed={selected?.tmdb_id === result.id}>
        {result.poster_path ? <img src={tmdbPosterUrl(result.poster_path, 'w92')} alt="" loading="lazy" referrerPolicy="no-referrer" /> : <span className="media-lookup-no-poster">暂无海报</span>}
        <span>{result.title}<small>{result.year || '年份未知'}</small></span>
        {selected?.tmdb_id === result.id && <Check size={15} aria-hidden="true" />}
      </button>)}</div>
    </div>}
  </div>
}
