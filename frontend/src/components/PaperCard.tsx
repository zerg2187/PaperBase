import type { SearchResult } from '../types'

type PaperCardProps = {
  paper: SearchResult
  selected: boolean
  authRole: 'admin' | 'guest' | null
  onToggle: () => void
  onTagClick: () => void
  onDelete: () => void
  onCopyBibTeX: () => void
}

export function PaperCard({
  paper,
  selected,
  authRole,
  onToggle,
  onTagClick,
  onDelete,
  onCopyBibTeX,
}: PaperCardProps) {
  const mainTag = paper.tags && paper.tags.length > 0 ? paper.tags[0] : null
  const extraTags = paper.tags ? paper.tags.slice(1) : []

  return (
    <article
      onClick={onToggle}
      className={`paper-card ${selected ? 'selected' : ''}`}
    >
      <div className="card-header">
        <input
          type="checkbox"
          checked={selected}
          onChange={onToggle}
          onClick={(e) => e.stopPropagation()}
          className="card-checkbox"
        />
        {authRole === 'admin' && (
          <>
            <div className="card-tags">
              {extraTags.map(tag => (
                <span
                  key={tag.id}
                  className="card-tag"
                  style={{ backgroundColor: `${tag.color}20`, color: tag.color }}
                >
                  {tag.name}
                </span>
              ))}
            </div>
            <button
              onClick={(e) => { e.stopPropagation(); onTagClick() }}
              className="card-main-tag"
              title={mainTag ? `タグ: ${mainTag.name}` : 'タグ設定'}
              style={mainTag ? {
                backgroundColor: `${mainTag.color}20`,
                color: mainTag.color,
                borderColor: mainTag.color
              } : undefined}
            >
              {mainTag ? mainTag.name : 'タグなし'}
            </button>
          </>
        )}
      </div>

      <h3 className="card-title">{paper.title}</h3>
      <p className="card-authors">{paper.authors.join(', ')}</p>
      <p className="card-venue">
        <span className="venue-icon">📍</span>
        {paper.venue} ({paper.year})
      </p>
      {paper.similarity && (
        <div className="similarity-badge">
          類似度: {(paper.similarity * 100).toFixed(1)}%
        </div>
      )}
      <p className="card-abstract">
        {paper.abstract.length > 200
          ? paper.abstract.slice(0, 200) + '...'
          : paper.abstract}
      </p>

      <div className="card-actions" onClick={(e) => e.stopPropagation()}>
        <button onClick={onCopyBibTeX} className="action-btn action-copy" title="BibTeXコピー">
          <span>📋</span>
          BibTeX
        </button>
        <a
          href={`https://alphaxiv.org/abs/${paper.id}`}
          target="_blank"
          rel="noopener noreferrer"
          className="action-btn action-link"
        >
          <span>💬</span>
          AlphaXiv
        </a>
        <a
          href={`https://arxiv.org/abs/${paper.id}`}
          target="_blank"
          rel="noopener noreferrer"
          className="action-btn action-link"
        >
          <span>🔗</span>
          arXiv
        </a>
        {(authRole === 'admin' || paper.is_owned_by_me) && (
          <button onClick={onDelete} className="action-btn action-delete" title="削除">
            <span>🗑️</span>
            削除
          </button>
        )}
      </div>
    </article>
  )
}
