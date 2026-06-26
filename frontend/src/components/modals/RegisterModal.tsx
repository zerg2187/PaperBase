import { useState } from 'react'
import type { Tag } from '../../types'

type RegisterModalProps = {
  isOpen: boolean
  tags: Tag[]
  loading: boolean
  progress: { current: number; total: number } | null
  onClose: () => void
  onRegister: (ids: string[], tagIds: number[]) => Promise<void>
}

export function RegisterModal({ isOpen, tags, loading, progress, onClose, onRegister }: RegisterModalProps) {
  const [arxivIds, setArxivIds] = useState('')
  const [selectedTags, setSelectedTags] = useState<number[]>([])

  if (!isOpen) return null

  const toggleTag = (tagId: number) => {
    setSelectedTags(prev =>
      prev.includes(tagId)
        ? prev.filter(id => id !== tagId)
        : [...prev, tagId]
    )
  }

  const handleSubmit = async () => {
    const ids = arxivIds.split('\n').map(id => id.trim()).filter(id => id)
    if (ids.length === 0) return

    await onRegister(ids, selectedTags)
    setArxivIds('')
    setSelectedTags([])
    onClose()
  }

  return (
    <div className="modal-overlay" onClick={loading ? undefined : onClose}>
      <div className="modal-container" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>論文登録</h2>
          <button onClick={onClose} disabled={loading} className="modal-close-btn">×</button>
        </div>
        <form onSubmit={(e) => { e.preventDefault(); handleSubmit() }} className="modal-body">
          <div className="form-group">
            <label htmlFor="arxiv-ids" className="form-label">
              arXiv ID <span className="form-hint">（複数の場合は改行区切り）</span>
            </label>
            <textarea
              id="arxiv-ids"
              value={arxivIds}
              onChange={(e) => setArxivIds(e.target.value)}
              placeholder="例:&#10;2406.11717&#10;2310.12345&#10;2312.67890"
              disabled={loading}
              rows={5}
              className="form-textarea"
            />
          </div>

          <div className="form-group">
            <label className="form-label">タグ（オプション）</label>
            <div className="tag-selection-area">
              {tags.map(tag => (
                <button
                  key={tag.id}
                  type="button"
                  onClick={() => toggleTag(tag.id)}
                  disabled={loading}
                  className={selectedTags.includes(tag.id) ? 'tag-select-btn selected' : 'tag-select-btn'}
                >
                  <span className="tag-color-dot" style={{ backgroundColor: tag.color }} />
                  {tag.name}
                </button>
              ))}
              {tags.length === 0 && (
                <span className="no-tags-hint">タグがありません。先にタグを作成してください。</span>
              )}
            </div>
          </div>

          <div className="modal-footer">
            <button type="button" onClick={onClose} disabled={loading} className="btn-secondary">
              キャンセル
            </button>
            <button type="submit" disabled={loading || !arxivIds.trim()} className="btn-primary">
              {loading ? `登録中... ${progress ? `${progress.current}/${progress.total}` : ''}` : '登録'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
