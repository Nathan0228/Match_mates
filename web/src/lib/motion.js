const KEY = 'jianshan-reduce-motion'

export function readReducedMotion() {
  try { return window.localStorage.getItem(KEY) === 'true' }
  catch { return false }
}

export function setReducedMotion(enabled) {
  document.documentElement.dataset.reduceMotion = enabled ? 'true' : 'false'
  try { window.localStorage.setItem(KEY, String(enabled)) }
  catch { /* The preference still applies for this page when storage is unavailable. */ }
}

export function applyStoredMotionPreference() {
  document.documentElement.dataset.reduceMotion = readReducedMotion() ? 'true' : 'false'
}

export function preferredScrollBehavior() {
  return document.documentElement.dataset.reduceMotion === 'true' || window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth'
}
