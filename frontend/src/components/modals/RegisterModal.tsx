import { useState } from 'react'
import type { Tag } from '../../types'
import { randomColor } from '../../utils/randomColor'

type RegisterModalProps = {
  isOpen: boolean
  authRole: 'admin' | 'guest' | null
  tags: Tag[]
  loading: boolean
  progress: { current: number; total: number } | null
  onClose: () => void
  onRegister: (ids: string[], tagIds: number[]) => Promise<void>
  onCreateTag: (name: string, color: string) => Promise<Tag>
}

export function RegisterModal({ isOpen, authRole, tags, loading, progress, onClose, onRegister, onCreateTag }: RegisterModalProps) {
  const [arxivIds, setArxivIds] = useState('')
  const [selectedTags, setSelectedTags] = useState<number[]>([])
  const [newTagName, setNewTagName] = useState('')
  const [newTagColor, setNewTagColor] = useState(randomColor())
  const [creatingTag, setCreatingTag] = useState(false)

  if (!isOpen) return null

  const toggleTag = (tagId: number) => {
    setSelectedTags(prev =>
      prev.includes(tagId)
        ? prev.filter(id => id !== tagId)
        : [...prev, tagId]
    )
  }

  const handleCreateTag = async () => {
    if (!newTagName.trim() || creatingTag) return

    setCreatingTag(true)
    try {
      const tag = await onCreateTag(newTagName.trim(), newTagColor)
      setSelectedTags(prev => [...prev, tag.id])
      setNewTagName('')
      setNewTagColor(randomColor())
    } catch (err) {
      console.error('タグ作成エラー:', err)
    } finally {
      setCreatingTag(false)
    }
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
        <div className="modal-body">
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

          {authRole === 'admin' && (
            <>
              <div className="form-group">
                <label className="form-label">タグ（オプション）</label>
                <div className="tag-selection-area">
                  {tags.map(tag => {
                    const isSelected = selectedTags.includes(tag.id)
                    return (
                      <button
                        key={tag.id}
                        type="button"
                        onClick={() => toggleTag(tag.id)}
                        disabled={loading}
                        aria-pressed={isSelected}
                        className={isSelected ? 'tag-select-btn selected' : 'tag-select-btn'}
                        style={isSelected ? {
                          borderColor: tag.color,
                          backgroundColor: `${tag.color}20`,
                          color: tag.color,
                        } : undefined}
                      >
                        <span className="tag-select-check">{isSelected ? '✓' : ''}</span>
                        <span className="tag-color-dot" style={{ backgroundColor: tag.color }} />
                        {tag.name}
                      </button>
                    )
                  })}
                  {tags.length === 0 && (
                    <span className="no-tags-hint">タグがありません。下のフォームから作成できます。</span>
                  )}
                </div>
              </div>

              <div className="form-group">
                <label className="form-label">新しいタグを作成</label>
                <div className="new-tag-form-row">
                  <input
                    type="text"
                    value={newTagName}
                    onChange={(e) => setNewTagName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault()
                        handleCreateTag()
                      }
                    }}
                    placeholder="タグ名"
                    disabled={loading || creatingTag}
                    className="form-input-small"
                  />
                  <div className="color-input-wrapper">
                    <input
                      type="color"
                      value={newTagColor}
                      onChange={(e) => setNewTagColor(e.target.value)}
                      disabled={loading || creatingTag}
                      className="color-input-small"
                    />
                    <span className="color-label">{newTagColor}</span>
                  </div>
                  <button
                    type="button"
                    onClick={handleCreateTag}
                    disabled={loading || creatingTag || !newTagName.trim()}
                    className="btn-small"
                  >
                    追加
                  </button>
                </div>
              </div>
            </>
          )}

          <div className="modal-footer">
            <button type="button" onClick={onClose} disabled={loading} className="btn-secondary">
              キャンセル
            </button>
            <button type="button" onClick={handleSubmit} disabled={loading || !arxivIds.trim()} className="btn-primary">
              {loading ? `登録中... ${progress ? `${progress.current}/${progress.total}` : ''}` : '登録'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
