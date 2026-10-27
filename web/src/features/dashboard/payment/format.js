export function money(cents) { return `¥${(cents / 100).toFixed(2)}` }
export function dateTime(seconds) { return seconds ? new Date(seconds * 1000).toLocaleString('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }) : '—' }
export function channelName(channel) { return channel === 'wechat' ? '微信' : channel === 'alipay' ? '支付宝' : channel }
