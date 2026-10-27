import { useCallback, useRef, useState } from 'react'
import EmptyState from '../../../components/EmptyState'
import ChoiceGroup from '../../../components/ChoiceGroup'
import { request } from '../../../lib/api'
import useAutoRefresh from '../../../hooks/useAutoRefresh'
import UserDetail from './UserDetail'
import formatTime from './formatTime'

const filters = [
  { value: 'all', label: '全部用户' },
  { value: 'active', label: '可参与匹配' },
  { value: 'inactive', label: '不可参与匹配' },
  { value: 'unpaid', label: '会费未完成' },
  { value: 'card_none', label: '未提交学生证' },
  { value: 'card_pending', label: '证件待审核' },
  { value: 'suspended', label: '已停用' },
]

export default function UserDirectory({ notify, onChanged }) {
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState('all')
  const [page, setPage] = useState(1)
  const [result, setResult] = useState({ users: [], total: 0, page_size: 20 })
  const [selectedId, setSelectedId] = useState(null)
  const [detail, setDetail] = useState(null)
  const [listLoading, setListLoading] = useState(true)
  const [detailLoading, setDetailLoading] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const listRequestId = useRef(0)
  const detailRequestId = useRef(0)

  const loadList = useCallback(async (query = search, selectedFilter = filter, currentPage = page) => {
    const requestId = ++listRequestId.current
    try {
      const data = await request(`/admin/users?q=${encodeURIComponent(query)}&filter=${selectedFilter}&page=${currentPage}`)
      if (requestId !== listRequestId.current) return
      setResult(data)
      setSelectedId(current => data.users.some(item => item.id === current) ? current : data.users[0]?.id || null)
      setError('')
    } catch (err) { if (requestId === listRequestId.current) setError(err.message) }
    finally { if (requestId === listRequestId.current) setListLoading(false) }
  }, [search, filter, page])

  const loadDetail = useCallback(async () => {
    const requestId = ++detailRequestId.current
    if (!selectedId) { setDetail(null); return }
    setDetailLoading(true)
    try {
      const data = await request(`/admin/users/${encodeURIComponent(selectedId)}`)
      if (requestId !== detailRequestId.current) return
      setDetail(data)
      setError('')
    } catch (err) { if (requestId === detailRequestId.current) { setDetail(null); setError(err.message) } }
    finally { if (requestId === detailRequestId.current) setDetailLoading(false) }
  }, [selectedId])

  useAutoRefresh(loadList)
  useAutoRefresh(loadDetail, Boolean(selectedId))

  async function changeStatus(suspended, reason) {
    if (!selectedId || busy) return false
    setBusy(true); setError('')
    try {
      await request(`/admin/users/${encodeURIComponent(selectedId)}/status`, { method: 'POST', body: JSON.stringify({ suspended, reason }) })
      await Promise.all([loadList(), loadDetail(), onChanged()])
      notify(suspended ? '账号已停用，登录会话已失效' : '账号已恢复')
      return true
    } catch (err) { setError(err.message); return false }
    finally { setBusy(false) }
  }

  function submitSearch(event) {
    event.preventDefault()
    const query = searchInput.trim()
    setPage(1)
    setSearch(query)
    setListLoading(true)
    void loadList(query, filter, 1)
  }

  return <div className="admin-user-page">
    <p className="admin-history-intro">查看用户入驻状态与资料，按邮箱、昵称或用户 ID 检索。账号停用与恢复会留存操作记录。</p>
    <form className="admin-user-toolbar" onSubmit={submitSearch}><label className="admin-search"><input value={searchInput} onChange={event => setSearchInput(event.target.value)} maxLength={80} placeholder="搜索邮箱、昵称或用户 ID" aria-label="搜索用户" /></label><button type="submit">搜索</button><div className="admin-user-filter"><ChoiceGroup label="筛选用户" hint="不可参与匹配包含未完成条件或已停用账号" options={filters} value={filter} onChange={value => { setFilter(value); setPage(1); setListLoading(true); void loadList(search, value, 1) }} required collapseOnSelect /></div></form>
    {error && <div className="form-error" role="alert">{error}</div>}
    <div className="admin-user-count">找到 {result.total} 位用户 · 第 {page} 页</div>
    {listLoading ? <div className="admin-loading">正在加载用户…</div> : result.users.length ? <div className="admin-review-layout">
      <div className="admin-review-list admin-user-list">{result.users.map(person => <button type="button" key={person.id} className={person.id === selectedId ? 'selected' : ''} onClick={() => { if (person.id !== selectedId) { setSelectedId(person.id); setDetail(null) } }}><span className="admin-review-avatar">{person.nickname?.slice(0, 1) || '同'}</span><span className="admin-review-name"><strong>{person.nickname}{person.is_admin ? ' · 管理员' : ''}</strong><small>{person.email}</small><em>{person.suspended_at ? '已停用' : person.paid ? '会费已完成' : '会费未完成'} · {formatTime(person.created_at)}</em></span></button>)}</div>
      <UserDetail key={selectedId} detail={detail?.user?.id === selectedId ? detail : null} loading={detailLoading} busy={busy} onStatusChange={changeStatus} />
    </div> : <EmptyState title="没有符合条件的用户" text="调整搜索内容或筛选条件后重试。" />}
    <div className="admin-user-pagination"><button type="button" disabled={page <= 1 || listLoading} onClick={() => setPage(value => value - 1)}>上一页</button><span>{page} / {Math.max(1, Math.ceil(result.total / result.page_size))}</span><button type="button" disabled={page * result.page_size >= result.total || listLoading} onClick={() => setPage(value => value + 1)}>下一页</button></div>
  </div>
}
