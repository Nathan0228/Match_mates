import { BookOpen, Film, Gamepad2, Music2, Plus, Trash2, Tv } from 'lucide-react'
import MediaShelf from '../../components/MediaShelf'
import { mediaCategories } from '../../data/media'
import TMDBLookup from './TMDBLookup'

const icons = { music: Music2, movies: Film, books: BookOpen, shows: Tv, games: Gamepad2 }

export default function MediaCollectionsEditor({ media, onChange }) {
  function update(key, index, field, value) {
    onChange(current => ({ ...current, [key]: current[key].map((item, itemIndex) => itemIndex === index ? {
      ...item,
      [field]: value,
      ...(field === 'title' ? { tmdb_id: 0, poster_path: '' } : {}),
    } : item) }))
  }

  function selectMatch(key, index, match) {
    onChange(current => ({ ...current, [key]: current[key].map((item, itemIndex) => itemIndex === index ? {
      ...item,
      tmdb_id: match?.id || 0,
      poster_path: match?.poster_path || '',
    } : item) }))
  }

  function add(key) {
    onChange(current => current[key].length >= 5 ? current : { ...current, [key]: [...current[key], { title: '', creator: '' }] })
  }

  function remove(key, index) {
    onChange(current => ({ ...current, [key]: current[key].filter((_, itemIndex) => itemIndex !== index) }))
  }

  return <section className="media-editor" aria-labelledby="media-editor-title">
    <h3 id="media-editor-title">我的作品清单</h3>
    <p className="field-hint">用几部作品介绍现在的你。每类最多 5 项，想分享多少都由你决定。</p>
    <div className="media-editor-categories">
      {mediaCategories.map(category => {
        const Icon = icons[category.key]
        const items = media[category.key]
        const previewItems = items.filter(item => item.title.trim())
        return <details className="media-editor-category" key={category.key}>
          <summary><Icon size={18} aria-hidden="true" /><strong>{category.label}</strong><span>{items.length} / 5</span></summary>
          <div className="media-editor-body">
            {items.map((item, index) => <div className="media-editor-entry" key={index}>
              <div className="media-editor-row">
                <label>作品名称<input maxLength={80} value={item.title} onChange={event => update(category.key, index, 'title', event.target.value)} placeholder={category.example} /></label>
                <label>{category.creatorLabel}<input maxLength={60} value={item.creator} onChange={event => update(category.key, index, 'creator', event.target.value)} placeholder="选填" /></label>
                <button type="button" className="media-editor-remove" aria-label={`移除${category.label}第 ${index + 1} 项`} onClick={() => remove(category.key, index)}><Trash2 size={17} /></button>
              </div>
              {['movies', 'shows'].includes(category.key) && <TMDBLookup kind={category.key} title={item.title} selected={item} onSelect={match => selectMatch(category.key, index, match)} />}
            </div>)}
            <button className="media-editor-add" type="button" disabled={items.length >= 5} onClick={() => add(category.key)}><Plus size={16} />添加{category.key === 'books' ? '一本' : '一项'}</button>
            {previewItems.length > 0 && <div className="media-editor-preview"><span>展示预览</span><MediaShelf category={category} items={previewItems} /></div>}
          </div>
        </details>
      })}
    </div>
  </section>
}
