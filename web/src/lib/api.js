export const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

export async function request(path, options = {}) {
  const response = await fetch(`${API_BASE}/api${path}`, {
    credentials: 'include',
    cache: 'no-store',
    headers: options.body instanceof FormData ? undefined : { 'Content-Type': 'application/json' },
    ...options,
  })
  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    const error = new Error(data.error || '请求失败，请稍后再试')
    error.status = response.status
    throw error
  }
  if (options.method && options.method !== 'GET' && path !== '/logout' && !path.startsWith('/notifications') && typeof window !== 'undefined') {
    window.dispatchEvent(new Event('matchmates:data-changed'))
  }
  return data
}

export function avatarSrc(person) {
  return person.avatar_url ? `${API_BASE}${person.avatar_url}` : ''
}
