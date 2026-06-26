type PaginationProps = {
  currentPage: number
  hasNext: boolean
  onPrev: () => void
  onNext: () => void
}

export function Pagination({ currentPage, hasNext, onPrev, onNext }: PaginationProps) {
  return (
    <div className="pagination-section">
      <button
        onClick={onPrev}
        disabled={currentPage === 0}
        className="pagination-btn"
      >
        ← 前へ
      </button>
      <span className="pagination-info">{currentPage + 1}ページ目</span>
      <button
        onClick={onNext}
        disabled={!hasNext}
        className="pagination-btn"
      >
        次へ →
      </button>
    </div>
  )
}
