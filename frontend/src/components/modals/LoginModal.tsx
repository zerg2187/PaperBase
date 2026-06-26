import { useState } from 'react'

type LoginModalProps = {
  isOpen: boolean
  error: string | null
  loading: boolean
  onClose: () => void
  onLogin: (token: string) => Promise<void>
}

export function LoginModal({ isOpen, error, loading, onClose, onLogin }: LoginModalProps) {
  const [token, setToken] = useState('')

  if (!isOpen) return null

  const handleSubmit = async () => {
    if (!token.trim()) return
    await onLogin(token.trim())
    setToken('')
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-container" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>管理者ログイン</h2>
          <button onClick={onClose} className="modal-close-btn">×</button>
        </div>
        <form onSubmit={(e) => { e.preventDefault(); handleSubmit() }} className="modal-body">
          <div className="form-group">
            <label htmlFor="admin-token" className="form-label">管理者トークン</label>
            <input
              id="admin-token"
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="ADMIN_SECRET_TOKEN を入力"
              disabled={loading}
              className="form-input"
              autoFocus
            />
            <p className="form-help">
              管理者としてログインすると、論文の無制限登録・一括削除ができます。
            </p>
          </div>
          {error && (
            <div className="form-error">⚠ {error}</div>
          )}
          <div className="modal-footer">
            <button type="button" onClick={onClose} className="btn-secondary">
              キャンセル
            </button>
            <button type="submit" disabled={loading || !token.trim()} className="btn-primary">
              {loading ? 'ログイン中...' : 'ログイン'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
