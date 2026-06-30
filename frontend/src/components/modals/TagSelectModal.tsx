import { useState, useEffect } from 'react'
import type { SearchResult, Tag } from '../../types'
import { randomColor } from '../../utils/randomColor'


type TagSelectModalProps = {
  paper: SearchResult
  tags: Tag[]
  onClose: () => void
  onSetTags: (paperId: string, tagIds: number[]) => Promise<void>
  onCreateTag: (name: string, color: string) => Promise<Tag>
}

export function TagSelectModal({ paper, tags, onClose, onSetTags, onCreateTag }: TagSelectModalProps) {
  const [selectedTags, setSelectedTags] = useState<number[]>([])
  const [newTagName, setNewTagName] = useState('')
  const [newTagColor, setNewTagColor] = useState(randomColor())
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    setSelectedTags(paper.tags?.map(t => t.id) || [])
  }, [paper])

  const toggleTag = (tagId: number) => {
    setSelectedTags(prev =>
      prev.includes(tagId)
        ? prev.filter(id => id !== tagId)
        : [...prev, tagId]
    )
  }

  const handleCreateTag = async () => {
    if (!newTagName.trim()) return

    setLoading(true)
    try {
      const newTag = await onCreateTag(newTagName.trim(), newTagColor)
      setSelectedTags(prev => [...prev, newTag.id])
      setNewTagName('')
      setNewTagColor(randomColor())
    } catch (err) {
      console.error('タグ作成エラー:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async () => {
    await onSetTags(paper.id, selectedTags)
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-container" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>タグ設定</h2>
          <button onClick={onClose} className="modal-close-btn">×</button>
        </div>
        <div className="modal-body">
          <p className="paper-title-preview">{paper.title}</p>

          <div className="tag-selection-section">
            <h4 className="section-label">タグを選択:</h4>
            <div className="tag-grid">
              {tags.map(tag => (
                <button
                  key={tag.id}
                  onClick={() => toggleTag(tag.id)}
                  className={`tag-card ${selectedTags.includes(tag.id) ? 'selected' : ''}`}
                  style={selectedTags.includes(tag.id) ? {
                    borderColor: tag.color,
                    backgroundColor: `${tag.color}15`
                  } : undefined}
                >
                  <span className="tag-color-dot" style={{ backgroundColor: tag.color }} />
                  {tag.name}
                </button>
              ))}
            </div>
          </div>

          <form onSubmit={(e) => { e.preventDefault(); handleCreateTag() }} className="new-tag-section">
            <h4 className="section-label">新しいタグを作成:</h4>
            <div className="new-tag-form-row">
              <input
                type="text"
                value={newTagName}
                onChange={(e) => setNewTagName(e.target.value)}
                placeholder="タグ名"
                disabled={loading}
                className="form-input-small"
              />
              <div className="color-input-wrapper">
                <input
                  type="color"
                  value={newTagColor}
                  onChange={(e) => setNewTagColor(e.target.value)}
                  disabled={loading}
                  className="color-input-small"
                />
                <span className="color-label">{newTagColor}</span>
              </div>
              <button type="submit" disabled={loading || !newTagName.trim()} className="btn-small">
                追加
              </button>
            </div>
          </form>
        </div>
        <div className="modal-footer">
          <button onClick={onClose} className="btn-secondary">
            キャンセル
          </button>
          <button onClick={handleSave} className="btn-primary">
            保存
          </button>
        </div>
      </div>
    </div>
  )
}
