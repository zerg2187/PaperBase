import { useState } from 'react'
import type { Tag } from '../../types'
import { TagEditModal } from './TagEditModal'

type TagManagerModalProps = {
  isOpen: boolean
  tags: Tag[]
  loading: boolean
  onClose: () => void
  onCreateTag: (name: string, color: string) => Promise<void>
  onUpdateTag: (id: number, name: string, color: string) => Promise<void>
  onDeleteTag: (id: number, name: string) => Promise<void>
  onBulkDeleteTags: (ids: number[]) => Promise<void>
}

export function TagManagerModal({
  isOpen,
  tags,
  loading,
  onClose,
  onCreateTag,
  onUpdateTag,
  onDeleteTag,
  onBulkDeleteTags,
}: TagManagerModalProps) {
  const [newTagName, setNewTagName] = useState('')
  const [newTagColor, setNewTagColor] = useState('#6366f1')
  const [selectedTags, setSelectedTags] = useState<Set<number>>(new Set())
  const [editingTag, setEditingTag] = useState<Tag | null>(null)

  if (!isOpen) return null

  const toggleTagSelection = (tagId: number) => {
    setSelectedTags(prev => {
      const next = new Set(prev)
      if (next.has(tagId)) next.delete(tagId)
      else next.add(tagId)
      return next
    })
  }

  const toggleSelectAllTags = () => {
    if (selectedTags.size === tags.length) {
      setSelectedTags(new Set())
    } else {
      setSelectedTags(new Set(tags.map(t => t.id)))
    }
  }

  const handleCreateTag = async () => {
    if (!newTagName.trim()) return
    await onCreateTag(newTagName.trim(), newTagColor)
    setNewTagName('')
    setNewTagColor('#6366f1')
  }

  const handleBulkDeleteTags = async () => {
    if (selectedTags.size === 0) return
    const selectedTagNames = tags.filter(t => selectedTags.has(t.id)).map(t => t.name)
    if (!confirm(`${selectedTags.size}件のタグを削除しますか?\n\n${selectedTagNames.join(', ')}`)) return
    await onBulkDeleteTags(Array.from(selectedTags))
    setSelectedTags(new Set())
  }

  return (
    <>
      <div className="modal-overlay" onClick={onClose}>
        <div className="modal-container" onClick={(e) => e.stopPropagation()}>
          <div className="modal-header">
            <h2>タグ管理</h2>
            <button onClick={onClose} className="modal-close-btn">×</button>
          </div>
          <div className="modal-body">
            <form onSubmit={(e) => { e.preventDefault(); handleCreateTag() }} className="new-tag-form">
              <h4 className="section-label">新しいタグを作成</h4>
              <div className="form-row">
                <input
                  id="tag-name"
                  type="text"
                  value={newTagName}
                  onChange={(e) => setNewTagName(e.target.value)}
                  placeholder="例: Transformer, LLM"
                  disabled={loading}
                  className="form-input-small"
                />
                <div className="color-input-wrapper">
                  <input
                    id="tag-color"
                    type="color"
                    value={newTagColor}
                    onChange={(e) => setNewTagColor(e.target.value)}
                    disabled={loading}
                    className="color-input-small"
                  />
                  <span className="color-label">{newTagColor}</span>
                </div>
                <button type="submit" disabled={loading || !newTagName.trim()} className="btn-small">
                  作成
                </button>
              </div>
            </form>

            {tags.length > 0 && (
              <div className="existing-tags-section">
                <div className="existing-tags-header">
                  <h4 className="section-label">既存のタグ</h4>
                  <label className="checkbox-label-small">
                    <input
                      type="checkbox"
                      checked={selectedTags.size === tags.length && tags.length > 0}
                      onChange={toggleSelectAllTags}
                      className="checkbox-input-small"
                    />
                    <span className="checkbox-text-small">全選択 ({selectedTags.size}/{tags.length})</span>
                  </label>
                </div>
                <div className="existing-tags-list">
                  {tags.map(tag => (
                    <div key={tag.id} className={`existing-tag-item ${selectedTags.has(tag.id) ? 'selected' : ''}`}>
                      <input
                        type="checkbox"
                        checked={selectedTags.has(tag.id)}
                        onChange={() => toggleTagSelection(tag.id)}
                        className="checkbox-input-small"
                      />
                      <span className="tag-color-dot" style={{ backgroundColor: tag.color }} />
                      <span className="tag-name">{tag.name}</span>
                      <div className="tag-actions">
                        <button
                          onClick={() => setEditingTag(tag)}
                          className="tag-action-btn"
                          title="編集"
                        >
                          ✏️
                        </button>
                        <button
                          onClick={() => onDeleteTag(tag.id, tag.name)}
                          className="tag-action-btn tag-delete-btn"
                          title="削除"
                        >
                          🗑️
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
                {selectedTags.size > 0 && (
                  <div className="bulk-action-bar">
                    <span className="bulk-action-text">{selectedTags.size}件選択中</span>
                    <button onClick={handleBulkDeleteTags} className="btn-small btn-danger">
                      🗑️ 選択したタグを削除
                    </button>
                  </div>
                )}
              </div>
            )}
          </div>
          <div className="modal-footer">
            <button onClick={onClose} className="btn-secondary">
              閉じる
            </button>
          </div>
        </div>
      </div>

      {editingTag && (
        <TagEditModal
          tag={editingTag}
          loading={loading}
          onClose={() => setEditingTag(null)}
          onSave={async (id, name, color) => {
            await onUpdateTag(id, name, color)
            setEditingTag(null)
          }}
        />
      )}
    </>
  )
}
