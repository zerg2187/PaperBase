import type { Tag } from '../types'

type TagFilterProps = {
  tags: Tag[]
  selectedTagFilter: number | null
  onSelect: (tagId: number | null) => void
}

export function TagFilter({ tags, selectedTagFilter, onSelect }: TagFilterProps) {
  if (tags.length === 0) return null

  return (
    <div className="tag-filter-section">
      <span className="tag-filter-label">タグで絞り込み:</span>
      <button
        onClick={() => onSelect(null)}
        className={`tag-filter-chip ${selectedTagFilter === null ? 'active' : ''}`}
      >
        すべて
      </button>
      {tags.map(tag => (
        <button
          key={tag.id}
          onClick={() => onSelect(tag.id)}
          className={`tag-filter-chip ${selectedTagFilter === tag.id ? 'active' : ''}`}
          style={selectedTagFilter === tag.id ? {
            borderColor: tag.color,
            backgroundColor: `${tag.color}15`,
            color: tag.color,
            fontWeight: '600'
          } : undefined}
        >
          <span className="tag-color-dot" style={{ backgroundColor: tag.color }} />
          {tag.name}
        </button>
      ))}
    </div>
  )
}
