import { useState } from 'react'
import type { Tag } from '../../types'

type TagEditModalProps = {
  tag: Tag | null
  loading: boolean
  onClose: () => void
  onSave: (id: number, name: string, color: string) => Promise<void>
}

export function TagEditModal({ tag, loading, onClose, onSave }: TagEditModalProps) {
  const [name, setName] = useState(tag?.name ?? '')
  const [color, setColor] = useState(tag?.color ?? '#6366f1')

  if (!tag) return null

  const handleSubmit = async () => {
    if (!name.trim()) return
    await onSave(tag.id, name.trim(), color)
    onClose()
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-container" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>タグ編集</h2>
          <button onClick={onClose} className="modal-close-btn">×</button>
        </div>
        <form onSubmit={(e) => { e.preventDefault(); handleSubmit() }} className="modal-body">
          <div className="form-group">
            <label htmlFor="edit-tag-name" className="form-label">タグ名</label>
            <input
              id="edit-tag-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="タグ名"
              disabled={loading}
              className="form-input"
            />
          </div>
          <div className="form-group">
            <label htmlFor="edit-tag-color" className="form-label">色</label>
            <div className="color-input-wrapper">
              <input
                id="edit-tag-color"
                type="color"
                value={color}
                onChange={(e) => setColor(e.target.value)}
                disabled={loading}
                className="color-input-small"
              />
              <span className="color-label">{color}</span>
            </div>
          </div>
          <div className="modal-footer">
            <button type="button" onClick={onClose} className="btn-secondary">
              キャンセル
            </button>
            <button type="submit" disabled={loading || !name.trim()} className="btn-primary">
              {loading ? '保存中...' : '保存'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
