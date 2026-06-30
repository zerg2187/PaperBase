import type { SearchResult } from '../types'

type PaperListItemProps = {
  paper: SearchResult
  active: boolean
  inCart: boolean
  authRole: 'admin' | 'guest' | null
  onClick: () => void
  onToggleCart: () => void
}

export function PaperListItem({
  paper,
  active,
  inCart,
  authRole,
  onClick,
  onToggleCart,
}: PaperListItemProps) {
  const mainTag = paper.tags && paper.tags.length > 0 ? paper.tags[0] : null
  const extraTags = paper.tags ? paper.tags.slice(1) : []

  return (
    <article
      onClick={onClick}
      className={`paper-list-item ${active ? 'active' : ''}`}
    >
      <div className="paper-list-item-main">
        <h3 className="paper-list-title">{paper.title}</h3>
        {paper.similarity !== undefined && (
          <span className="paper-list-similarity">
            {(paper.similarity * 100).toFixed(0)}%
          </span>
        )}
      </div>

      {authRole === 'admin' && (
        <div className="paper-list-tags">
          {extraTags.map(tag => (
            <span
              key={tag.id}
              className="card-tag"
              style={{ backgroundColor: `${tag.color}20`, color: tag.color }}
            >
              {tag.name}
            </span>
          ))}
          {mainTag && (
            <span
              className="card-tag"
              style={{
                backgroundColor: `${mainTag.color}20`,
                color: mainTag.color,
                border: `1px solid ${mainTag.color}40`,
              }}
            >
              {mainTag.name}
            </span>
          )}
        </div>
      )}

      <button
        onClick={(e) => { e.stopPropagation(); onToggleCart() }}
        className={`paper-list-cart-btn ${inCart ? 'in-cart' : ''}`}
      >
        {inCart ? '🛒 カートに入っています' : '＋ カートに入れる'}
      </button>
    </article>
  )
}
