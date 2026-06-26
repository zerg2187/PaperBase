import type { SearchResult } from '../types'
import { PaperCard } from './PaperCard'
import { Pagination } from './Pagination'

type PaperListProps = {
  papers: SearchResult[]
  selectedPapers: Set<string>
  authRole: 'admin' | 'guest' | null
  loading: boolean
  showPagination: boolean
  currentPage: number
  hasNext: boolean
  onToggle: (id: string) => void
  onSelectAll: () => void
  onPrevPage: () => void
  onNextPage: () => void
  onTagClick: (paper: SearchResult) => void
  onDelete: (paper: SearchResult) => void
  onCopyBibTeX: (bibtex: string, title: string) => void
  onRegisterClick: () => void
}

export function PaperList({
  papers,
  selectedPapers,
  authRole,
  loading,
  showPagination,
  currentPage,
  hasNext,
  onToggle,
  onSelectAll,
  onPrevPage,
  onNextPage,
  onTagClick,
  onDelete,
  onCopyBibTeX,
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

          <div className="paper-grid">
            {papers.map(paper => (
              <PaperCard
                key={paper.id}
                paper={paper}
                selected={selectedPapers.has(paper.id)}
                authRole={authRole}
                onToggle={() => onToggle(paper.id)}
                onTagClick={() => onTagClick(paper)}
                onDelete={() => onDelete(paper)}
                onCopyBibTeX={() => onCopyBibTeX(paper.bibtex, paper.title)}
              />
            ))}
          </div>

          {showPagination && (
            <Pagination
              currentPage={currentPage}
              hasNext={hasNext}
              onPrev={onPrevPage}
              onNext={onNextPage}
            />
          )}
        </>
      )}
    </>
  )
}
