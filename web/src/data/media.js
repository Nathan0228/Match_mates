import { API_BASE } from '../lib/api'

export const mediaCategories = [
  { key: 'music', label: '最近爱听的音乐', creatorLabel: '歌手 / 乐队', example: '例如：想见你 · 八三夭', kind: 'record', mark: 'ON REPEAT' },
  { key: 'movies', label: '爱看的电影', creatorLabel: '导演 / 主演', example: '例如：海街日记', kind: 'poster', mark: 'FILM' },
  { key: 'books', label: '珍藏书籍', creatorLabel: '作者', example: '例如：也许你该找个人聊聊', kind: 'book', mark: 'BOOKS' },
  { key: 'shows', label: '追剧中', creatorLabel: '导演 / 主演', example: '例如：请回答1988', kind: 'poster', mark: 'WATCHING' },
  { key: 'games', label: '最近玩的游戏', creatorLabel: '平台 / 类型', example: '例如：双人成行', kind: 'poster', mark: 'PLAYING' },
]

export function normalizeMedia(media) {
  return Object.fromEntries(mediaCategories.map(({ key }) => [
    key,
    Array.isArray(media?.[key]) ? media[key].map(item => ({
      title: item?.title || '',
      creator: item?.creator || '',
      ...(['movies', 'shows'].includes(key) ? { tmdb_id: Number.isInteger(item?.tmdb_id) ? item.tmdb_id : 0, poster_path: item?.poster_path || '' } : {}),
    })) : [],
  ]))
}

export function tmdbPosterUrl(path, size = 'w342') {
	return /^\/[A-Za-z0-9_-]+\.(?:jpg|jpeg|png|webp)$/.test(path || '') && ['w92', 'w342'].includes(size)
		? `${API_BASE}/api/media/poster/${size}${path}` : ''
}
