import type { Tag } from '../types'

type FilterBarProps = {
  isFiltered: boolean
  selectedTagFilter: number | null
  tags: Tag[]
  count: number
  onClear: () => void
}

export function FilterBar({ isFiltered, selectedTagFilter, tags, count, onClear }: FilterBarProps) {
  if (!isFiltered && selectedTagFilter === null) return null

  return (
    <div className={`filter-bar ${selectedTagFilter !== null ? 'tag-active' : ''}`}>
      <span className="filter-label">
        {selectedTagFilter !== null ? (
          <>タグ: {tags.find(t => t.id === selectedTagFilter)?.name} {count}件</>
        ) : (
          <>検索結果: {count}件</>
        )}
      </span>
      <button onClick={onClear} className="filter-clear-btn">✕ クリア</button>
    </div>
  )
}
