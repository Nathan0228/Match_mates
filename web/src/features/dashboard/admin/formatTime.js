export default function formatTime(seconds) {
  return seconds ? new Date(seconds * 1000).toLocaleString('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }) : '时间未记录'
}
