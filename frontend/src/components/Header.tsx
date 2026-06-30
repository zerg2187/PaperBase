type HeaderProps = {
  paperCount: number
  selectedCount: number
  authRole: 'admin' | 'guest' | null
  guestRemainingCount: number | null
  registerLoading: boolean
  onLoginClick: () => void
  onLogout: () => void
  onSearchClick: () => void
  onRegisterClick: () => void
  onTagManageClick: () => void
  onExportBibTeX: () => void
  onBulkDelete: () => void
}

export function Header({
  paperCount,
  selectedCount,
  authRole,
  guestRemainingCount,
  registerLoading,
  onLoginClick,
  onLogout,
  onSearchClick,
  onRegisterClick,
  onTagManageClick,
  onExportBibTeX,
  onBulkDelete,
}: HeaderProps) {
  return (
    <header className="app-header">
      <div className="header-content">
        <div className="header-left">
          <h1 className="app-title">
            <span className="title-icon">📚</span>
            Paperbase
          </h1>
          <p className="app-subtitle">AI論文のセマンティック検索システム</p>
        </div>
        <div className="header-stats">
          <span className="stat-badge">
            <span className="stat-icon">📄</span>
            {paperCount}件
          </span>
          {selectedCount > 0 && (
            <span className="stat-badge selected">
              <span className="stat-icon">✓</span>
              {selectedCount}件選択中
            </span>
          )}
          {authRole === 'guest' && guestRemainingCount !== null && (
            <span className="stat-badge" title="ゲストは最大5件まで登録できます">
              <span className="stat-icon">🎫</span>
              残り{guestRemainingCount}件
            </span>
          )}
          <span className={`stat-badge ${authRole === 'admin' ? 'selected' : ''}`}>
            <span className="stat-icon">{authRole === 'admin' ? '🔐' : '👤'}</span>
            {authRole === 'admin' ? 'Admin' : 'Guest'}
          </span>
          {authRole === 'admin' ? (
            <button onClick={onLogout} className="nav-btn nav-btn-logout">
              ログアウト
            </button>
          ) : (
            <button onClick={onLoginClick} className="nav-btn nav-btn-login">
              管理者ログイン
            </button>
          )}
        </div>
      </div>

      <nav className="nav-bar">
        <button onClick={onSearchClick} className="nav-btn nav-btn-search">
          <span className="btn-icon">🔍</span>
          検索
        </button>
        <button
          onClick={onRegisterClick}
          disabled={registerLoading}
          className="nav-btn nav-btn-register"
        >
          <span className="btn-icon">➕</span>
          登録
        </button>
        {authRole === 'admin' && (
          <button onClick={onTagManageClick} className="nav-btn nav-btn-tag">
            <span className="btn-icon">🏷️</span>
            タグ管理
          </button>
        )}
        <div className="nav-divider" />
        {selectedCount > 0 && (
          <>
            <button onClick={onExportBibTeX} className="nav-btn nav-btn-export">
              <span className="btn-icon">📥</span>
              BibTeX出力
            </button>
            {authRole === 'admin' && (
              <button onClick={onBulkDelete} className="nav-btn nav-btn-delete">
                <span className="btn-icon">🗑️</span>
                一括削除
              </button>
            )}
          </>
        )}
      </nav>
    </header>
  )
}
