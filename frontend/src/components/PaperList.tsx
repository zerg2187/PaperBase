import type { SearchResult } from '../types'
import { PaperListItem } from './PaperListItem'

type PaperListProps = {
  papers: SearchResult[]
  selectedPapers: Set<string>
  activePaperId: string | null
  authRole: 'admin' | 'guest' | null
  loading: boolean
  isInCart: (id: string) => boolean
  onToggle: (id: string) => void
  onSelectAll: () => void
  onItemClick: (paper: SearchResult) => void
  onRegisterClick: () => void
}

export function PaperList({
  papers,
  selectedPapers,
  activePaperId,
  authRole,
  loading,
  isInCart,
  onToggle,
  onSelectAll,
  onItemClick,
  onRegisterClick,
}: PaperListProps) {
  const isAllSelected = papers.length > 0 && papers.every(p => selectedPapers.has(p.id))

  return (
    <>
      {loading && papers.length === 0 ? (
        <div className="loading-state">
          <div className="loading-spinner-large"></div>
          <p>読み込み中...</p>
        </div>
      ) : papers.length === 0 ? (
        <div className="empty-state">
          <div className="empty-icon">📭</div>
          <h2>論文がありません</h2>
          <p>検索ページから論文を登録してください</p>
          <button onClick={onRegisterClick} className="btn-primary">
            論文を登録
          </button>
        </div>
      ) : (
        <>
          <div className="select-all-section">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={isAllSelected}
                onChange={onSelectAll}
                className="checkbox-input"
              />
              <span className="checkbox-text">
                全選択 <span className="select-count">({selectedPapers.size}/{papers.length})</span>
              </span>
            </label>
          </div>

          <div className="paper-list">
            {papers.map(paper => (
              <PaperListItem
                key={paper.id}
                paper={paper}
                active={activePaperId === paper.id}
                inCart={isInCart(paper.id)}
                authRole={authRole}
                onClick={() => onItemClick(paper)}
                onToggleCart={() => onToggle(paper.id)}
              />
            ))}
          </div>
        </>
      )}
    </>
  )
}
