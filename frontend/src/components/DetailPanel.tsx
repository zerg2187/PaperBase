import type { SearchResult } from '../types'

type DetailPanelProps = {
  paper: SearchResult
  authRole: 'admin' | 'guest' | null
  onClose: () => void
  onTagClick: () => void
  onDelete: () => void
  onCopyBibTeX: () => void
}

export function DetailPanel({
  paper,
  authRole,
  onClose,
  onTagClick,
  onDelete,
  onCopyBibTeX,
}: DetailPanelProps) {
  const mainTag = paper.tags && paper.tags.length > 0 ? paper.tags[0] : null
  const extraTags = paper.tags ? paper.tags.slice(1) : []

  return (
    <aside className="detail-panel">
      <div className="detail-panel-header">
        <h2 className="detail-panel-title">詳細</h2>
        <button onClick={onClose} className="detail-panel-close">×</button>
      </div>

      <div className="detail-panel-content">
        {authRole === 'admin' && (
          <div className="detail-tags">
            {extraTags.map(tag => (
              <span
                key={tag.id}
                className="card-tag"
                style={{ backgroundColor: `${tag.color}20`, color: tag.color }}
              >
                {tag.name}
              </span>
            ))}
            <button
              onClick={onTagClick}
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
          </div>
        )}

        <h3 className="detail-title">{paper.title}</h3>

        <p className="detail-authors">{paper.authors.join(', ')}</p>

        <p className="detail-venue">
          <span className="venue-icon">📍</span>
          {paper.venue} ({paper.year})
        </p>

        {paper.similarity && (
          <div className="similarity-badge">
            類似度: {(paper.similarity * 100).toFixed(1)}%
          </div>
        )}

        <p className="detail-abstract">{paper.abstract}</p>

        <div className="detail-actions">
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
      </div>
    </aside>
  )
}
