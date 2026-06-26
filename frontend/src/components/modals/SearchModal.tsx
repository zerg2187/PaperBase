import { useState } from 'react'
import type { SearchMode } from '../../types'

const SIMILARITY_THRESHOLD = 0.30

type SearchModalProps = {
  isOpen: boolean
  loading: boolean
  onClose: () => void
  onSearch: (query: string, mode: SearchMode) => Promise<void>
}

export function SearchModal({ isOpen, loading, onClose, onSearch }: SearchModalProps) {
  const [query, setQuery] = useState('')
  const [mode, setMode] = useState<SearchMode>('semantic')

  if (!isOpen) return null

  const handleSubmit = async () => {
    if (!query.trim()) return
    await onSearch(query.trim(), mode)
    setQuery('')
    onClose()
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-container" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>論文検索</h2>
          <button onClick={onClose} className="modal-close-btn">×</button>
        </div>
        <form onSubmit={(e) => { e.preventDefault(); handleSubmit() }} className="modal-body">
          <div className="form-group">
            <label htmlFor="search-query" className="form-label">検索クエリ</label>
            <input
              id="search-query"
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="キーワードまたは自然言語"
              disabled={loading}
              className="form-input"
            />
          </div>
          <div className="form-group">
            <label htmlFor="search-mode" className="form-label">検索モード</label>
            <select
              id="search-mode"
              value={mode}
              onChange={(e) => setMode(e.target.value as SearchMode)}
              disabled={loading}
              className="form-select"
            >
              <option value="semantic">セマンティック（意味検索）</option>
              <option value="keyword">キーワード検索</option>
            </select>
            <p className="form-help">
              セマンティック検索では類似度{SIMILARITY_THRESHOLD * 100}%以上の論文を表示します
            </p>
          </div>
          <div className="modal-footer">
            <button type="button" onClick={onClose} className="btn-secondary">
              キャンセル
            </button>
            <button type="submit" disabled={loading || !query} className="btn-primary">
              {loading ? '検索中...' : '検索'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
