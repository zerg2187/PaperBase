import { useState, useEffect, useCallback } from 'react'
import './App.css'
import { registerPaper, searchPapers, getPapersPaginated, getAllTags, createTag, updateTag, deleteTag, setPaperTags, deletePapers } from './api'
import type { SearchResult, SearchMode, Tag } from './types'

// 定数
const SIMILARITY_THRESHOLD = 0.30 // 30%以上の類似度のみ表示
const ITEMS_PER_PAGE = 10

function App() {
  // 状態管理
  const [papers, setPapers] = useState<SearchResult[]>([])
  const [filteredPapers, setFilteredPapers] = useState<SearchResult[]>([])
  const [tags, setTags] = useState<Tag[]>([])
  const [selectedPapers, setSelectedPapers] = useState<Set<string>>(new Set())
  const [selectedTags, setSelectedTags] = useState<Set<number>>(new Set())

  // ページネーション
  const [currentPage, setCurrentPage] = useState(0)
  const [totalCount, setTotalCount] = useState(0)

  // 検索・フィルタ
  const [searchQuery, setSearchQuery] = useState('')
  const [searchMode, setSearchMode] = useState<SearchMode>('semantic')
  const [selectedTagFilter, setSelectedTagFilter] = useState<number | null>(null)
  const [isFiltered, setIsFiltered] = useState(false)

  // モーダル状態
  const [showRegisterModal, setShowRegisterModal] = useState(false)
  const [showSearchModal, setShowSearchModal] = useState(false)
  const [showTagModal, setShowTagModal] = useState(false)
  const [showTagEditModal, setShowTagEditModal] = useState(false)
  const [tagModalPaperId, setTagModalPaperId] = useState<string | null>(null)

  // ローディング・エラー・成功
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [successMessage, setSuccessMessage] = useState<string | null>(null)

  // 入力状態
  const [arxivIds, setArxivIds] = useState('')
  const [registerTags, setRegisterTags] = useState<number[]>([])
  const [newTagName, setNewTagName] = useState('')
  const [newTagColor, setNewTagColor] = useState('#6366f1')
  const [editingTagId, setEditingTagId] = useState<number | null>(null)
  const [editingTagName, setEditingTagName] = useState('')
  const [editingTagColor, setEditingTagColor] = useState('')

  // 論文を読み込む
  const loadPapers = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const offset = currentPage * ITEMS_PER_PAGE
      const data = await getPapersPaginated(offset, ITEMS_PER_PAGE, selectedTagFilter)
      setPapers(data)
      // 総数を計算（タグフィルター時はスキップ）
      if (selectedTagFilter === null && totalCount === 0) {
        setTotalCount(data.length >= ITEMS_PER_PAGE ? -1 : data.length)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '読み込みに失敗しました')
    } finally {
      setLoading(false)
    }
  }, [currentPage, selectedTagFilter, totalCount])

  // タグを読み込む
  const loadTags = async () => {
    try {
      const data = await getAllTags()
      setTags(data)
    } catch (err) {
      console.error('タグ読み込みエラー:', err)
    }
  }

  // 初期ロード
  useEffect(() => {
    loadPapers()
    loadTags()
  }, [loadPapers])

  // 検索実行
  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!searchQuery.trim()) return

    setLoading(true)
    setError(null)
    try {
      const data = await searchPapers(searchQuery, searchMode)

      const filtered = data.filter(p => {
        if (searchMode === 'semantic' && p.similarity !== undefined) {
          return p.similarity >= SIMILARITY_THRESHOLD
        }
        return true
      })

      const sorted = [...filtered].sort((a, b) => {
        if (a.similarity !== undefined && b.similarity !== undefined) {
          return b.similarity - a.similarity
        }
        return 0
      })

      setFilteredPapers(sorted)
      setIsFiltered(true)
      const newSelected = new Set(sorted.map(p => p.id))
      setSelectedPapers(newSelected)

      setSuccessMessage(`✓ ${sorted.length}件の論文が見つかりました`)
      setShowSearchModal(false) // 検索モーダルを閉じる
      setTimeout(() => setSuccessMessage(null), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : '検索に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  // フィルタ解除
  const clearFilter = () => {
    setFilteredPapers([])
    setIsFiltered(false)
    setSearchQuery('')
    setSelectedTagFilter(null)
    setSelectedPapers(new Set())
  }

  // 論文登録（一括対応）
  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault()
    const ids = arxivIds.split('\n').map(id => id.trim()).filter(id => id)

    if (ids.length === 0) return

    setLoading(true)
    setError(null)
    setSuccessMessage(null)

    const results = { success: 0, failed: 0, errors: [] as string[] }

    for (const id of ids) {
      try {
        await registerPaper(id)

        // タグ設定
        if (registerTags.length > 0) {
          await setPaperTags(id, registerTags)
        }

        results.success++
      } catch (err) {
        results.failed++
        results.errors.push(`${id}: ${err instanceof Error ? err.message : '登録失敗'}`)
      }
    }

    setArxivIds('')
    setRegisterTags([])
    setShowRegisterModal(false)

    if (results.success > 0) {
      setSuccessMessage(`✓ ${results.success}件の論文を登録しました`)
      loadPapers()
    }

    if (results.failed > 0) {
      setTimeout(() => {
        setError(`${results.failed}件失敗: ${results.errors.join(', ')}`)
        setTimeout(() => setError(null), 5000)
      }, 3000)
    } else {
      setTimeout(() => setSuccessMessage(null), 3000)
    }

    setLoading(false)
  }

  // 選択トグル
  const toggleSelection = (paperId: string) => {
    const newSelected = new Set(selectedPapers)
    if (newSelected.has(paperId)) {
      newSelected.delete(paperId)
    } else {
      newSelected.add(paperId)
    }
    setSelectedPapers(newSelected)
  }

  // 全選択/全解除
  const toggleSelectAll = () => {
    const currentPapers = isFiltered ? filteredPapers : papers
    if (selectedPapers.size === currentPapers.length) {
      setSelectedPapers(new Set())
    } else {
      setSelectedPapers(new Set(currentPapers.map(p => p.id)))
    }
  }

  // 個別BibTeXコピー
  const copyBibTeX = (bibtex: string, title: string) => {
    navigator.clipboard.writeText(bibtex)
    setSuccessMessage(`✓ 「${title.slice(0, 20)}...」のBibTeXをコピーしました`)
    setTimeout(() => setSuccessMessage(null), 3000)
  }

  // BibTeX一括エクスポート
  const exportBibTeX = () => {
    const currentPapers = isFiltered ? filteredPapers : papers
    const selectedPapersData = currentPapers.filter(p => selectedPapers.has(p.id))

    if (selectedPapersData.length === 0) {
      setError('論文が選択されていません')
      setTimeout(() => setError(null), 3000)
      return
    }

    const bibTeXContent = selectedPapersData.map(p => p.bibtex).join('\n\n')
    const blob = new Blob([bibTeXContent], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'papers.bib'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)

    setSuccessMessage(`✓ ${selectedPapersData.length}件のBibTeXをエクスポートしました`)
    setTimeout(() => setSuccessMessage(null), 3000)
  }

  // 一括削除
  const handleBulkDelete = async () => {
    const currentPapers = isFiltered ? filteredPapers : papers
    const selectedIds = Array.from(selectedPapers).filter(id =>
      currentPapers.some(p => p.id === id)
    )

    if (selectedIds.length === 0) {
      setError('論文が選択されていません')
      setTimeout(() => setError(null), 3000)
      return
    }

    if (!confirm(`${selectedIds.length}件の論文を削除しますか？`)) return

    setLoading(true)
    setError(null)

    try {
      await deletePapers(selectedIds)
      setSuccessMessage(`✓ ${selectedIds.length}件の論文を削除しました`)
      setSelectedPapers(new Set())

      if (isFiltered) {
        setFilteredPapers(filteredPapers.filter(p => !selectedIds.includes(p.id)))
      }

      loadPapers()
      setTimeout(() => setSuccessMessage(null), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : '削除に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  // タグ作成
  const handleCreateTag = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newTagName.trim()) return

    setLoading(true)
    try {
      await createTag(newTagName, newTagColor)
      setSuccessMessage(`✓ タグ「${newTagName}」を作成しました`)
      setNewTagName('')
      setNewTagColor('#6366f1')
      loadTags()
      setShowTagModal(false) // タグ管理モーダルを閉じる
      setTimeout(() => setSuccessMessage(null), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'タグ作成に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  // タグ削除
  const handleDeleteTag = async (tagId: number, tagName: string) => {
    if (!confirm(`タグ「${tagName}」を削除しますか？`)) return

    setLoading(true)
    try {
      await deleteTag(tagId)
      setSuccessMessage(`✓ タグ「${tagName}」を削除しました`)

      // 削除されたタグでフィルタしている場合は解除
      if (selectedTagFilter === tagId) {
        setSelectedTagFilter(null)
        setCurrentPage(0)
      }

      loadTags()
      loadPapers()
      setTimeout(() => setSuccessMessage(null), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'タグ削除に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  // タグ選択トグル
  const toggleTagSelection = (tagId: number) => {
    const newSelected = new Set(selectedTags)
    if (newSelected.has(tagId)) {
      newSelected.delete(tagId)
    } else {
      newSelected.add(tagId)
    }
    setSelectedTags(newSelected)
  }

  // タグ全選択/全解除
  const toggleSelectAllTags = () => {
    if (selectedTags.size === tags.length) {
      setSelectedTags(new Set())
    } else {
      setSelectedTags(new Set(tags.map(t => t.id)))
    }
  }

  // タグ一括削除
  const handleBulkDeleteTags = async () => {
    const selectedTagNames = tags.filter(t => selectedTags.has(t.id)).map(t => t.name)

    if (selectedTags.size === 0) {
      setError('タグが選択されていません')
      setTimeout(() => setError(null), 3000)
      return
    }

    if (!confirm(`${selectedTags.size}件のタグを削除しますか？\n\n${selectedTagNames.join(', ')}`)) return

    setLoading(true)
    setError(null)

    try {
      // 選択されたタグを削除
      for (const tagId of selectedTags) {
        await deleteTag(tagId)
      }

      setSuccessMessage(`✓ ${selectedTags.size}件のタグを削除しました`)
      setSelectedTags(new Set())

      // フィルタ中のタグが削除された場合は解除
      if (selectedTagFilter !== null && selectedTags.has(selectedTagFilter)) {
        setSelectedTagFilter(null)
        setCurrentPage(0)
      }

      loadTags()
      loadPapers()
      setTimeout(() => setSuccessMessage(null), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : '一括削除に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  // タグ編集開始
  const startEditTag = (tag: Tag) => {
    setEditingTagId(tag.id)
    setEditingTagName(tag.name)
    setEditingTagColor(tag.color)
    setShowTagEditModal(true)
  }

  // タグ編集キャンセル
  const cancelEditTag = () => {
    setEditingTagId(null)
    setEditingTagName('')
    setEditingTagColor('')
    setShowTagEditModal(false)
  }

  // タグ更新
  const handleUpdateTag = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingTagName.trim() || editingTagId === null) return

    setLoading(true)
    try {
      await updateTag(editingTagId, editingTagName, editingTagColor)
      setSuccessMessage(`✓ タグ「${editingTagName}」を更新しました`)
      setEditingTagId(null)
      setEditingTagName('')
      setEditingTagColor('')
      setShowTagEditModal(false)
      loadTags()
      setTimeout(() => setSuccessMessage(null), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'タグ更新に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  // タグモーダルを開く
  const openTagModal = (paperId: string) => {
    setTagModalPaperId(paperId)
    setShowTagModal(true)
  }

  // タグ設定
  const handleSetTags = async (tagIds: number[]) => {
    if (!tagModalPaperId) return

    setLoading(true)
    try {
      await setPaperTags(tagModalPaperId, tagIds)
      setSuccessMessage('✓ タグを更新しました')
      setShowTagModal(false)
      loadPapers()
      setTimeout(() => setSuccessMessage(null), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'タグ更新に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  // 登録時のタグ選択トグル
  const toggleRegisterTag = (tagId: number) => {
    if (registerTags.includes(tagId)) {
      setRegisterTags(registerTags.filter(id => id !== tagId))
    } else {
      setRegisterTags([...registerTags, tagId])
    }
  }

  // 現在表示している論文
  const displayPapers = (isFiltered && selectedTagFilter === null) ? filteredPapers : papers

  return (
    <div className="app">
      {/* ヘッダー */}
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
              {displayPapers.length}件
            </span>
            {selectedPapers.size > 0 && (
              <span className="stat-badge selected">
                <span className="stat-icon">✓</span>
                {selectedPapers.size}件選択中
              </span>
            )}
          </div>
        </div>

        {/* ナビゲーションバー */}
        <nav className="nav-bar">
          <button
            onClick={() => setShowSearchModal(true)}
            className="nav-btn nav-btn-search"
          >
            <span className="btn-icon">🔍</span>
            検索
          </button>
          <button
            onClick={() => setShowRegisterModal(true)}
            className="nav-btn nav-btn-register"
          >
            <span className="btn-icon">➕</span>
            登録
          </button>
          <button
            onClick={() => { setNewTagName(''); setSelectedTags(new Set()); setShowTagModal(true); }}
            className="nav-btn nav-btn-tag"
          >
            <span className="btn-icon">🏷️</span>
            タグ管理
          </button>
          <div className="nav-divider" />
          {selectedPapers.size > 0 && (
            <>
              <button onClick={exportBibTeX} className="nav-btn nav-btn-export">
                <span className="btn-icon">📥</span>
                BibTeX出力
              </button>
              <button onClick={handleBulkDelete} className="nav-btn nav-btn-delete">
                <span className="btn-icon">🗑️</span>
                一括削除
              </button>
            </>
          )}
        </nav>
      </header>

      {/* トースト通知 */}
      {successMessage && (
        <div className="toast toast-success">
          <span className="toast-icon">✓</span>
          {successMessage}
        </div>
      )}
      {error && (
        <div className="toast toast-error">
          <span className="toast-icon">⚠</span>
          {error}
        </div>
      )}

      {/* フィルタ表示 */}
      {(isFiltered || selectedTagFilter !== null) && (
        <div className={`filter-bar ${selectedTagFilter !== null ? 'tag-active' : ''}`}>
          <span className="filter-label">
            {selectedTagFilter !== null ? (
              <>タグ: {tags.find(t => t.id === selectedTagFilter)?.name} {papers.length}件</>
            ) : (
              <>検索結果: {filteredPapers.length}件</>
            )}
          </span>
          <button onClick={clearFilter} className="filter-clear-btn">✕ クリア</button>
        </div>
      )}

      {/* タグフィルタ */}
      {tags.length > 0 && (
        <div className="tag-filter-section">
          <span className="tag-filter-label">タグで絞り込み:</span>
          <button
            onClick={() => { setSelectedTagFilter(null); setCurrentPage(0); setIsFiltered(false); }}
            className={`tag-filter-chip ${selectedTagFilter === null ? 'active' : ''}`}
          >
            すべて
          </button>
          {tags.map(tag => (
            <button
              key={tag.id}
              onClick={() => { setSelectedTagFilter(tag.id); setCurrentPage(0); setIsFiltered(true); }}
              className={`tag-filter-chip ${selectedTagFilter === tag.id ? 'active' : ''}`}
              style={selectedTagFilter === tag.id ? {
                borderColor: tag.color,
                backgroundColor: `${tag.color}15`,
                color: tag.color,
                fontWeight: '600'
              } : undefined}
            >
              <span className="tag-color-dot" style={{ backgroundColor: tag.color }} />
              {tag.name}
            </button>
          ))}
        </div>
      )}

      {/* メインコンテンツ */}
      <main className="main-content">
        {loading && papers.length === 0 ? (
          <div className="loading-state">
            <div className="loading-spinner-large"></div>
            <p>読み込み中...</p>
          </div>
        ) : displayPapers.length === 0 ? (
          <div className="empty-state">
            <div className="empty-icon">📭</div>
            <h2>論文がありません</h2>
            <p>検索ページから論文を登録してください</p>
            <button onClick={() => setShowRegisterModal(true)} className="btn-primary">
              論文を登録
            </button>
          </div>
        ) : (
          <>
            {/* 全選択バー */}
            <div className="select-all-section">
              <label className="checkbox-label">
                <input
                  type="checkbox"
                  checked={selectedPapers.size === displayPapers.length && displayPapers.length > 0}
                  onChange={toggleSelectAll}
                  className="checkbox-input"
                />
                <span className="checkbox-text">
                  全選択 <span className="select-count">({selectedPapers.size}/{displayPapers.length})</span>
                </span>
              </label>
            </div>

            {/* 論文カードリスト */}
            <div className="paper-grid">
              {displayPapers.map((paper) => (
                <article
                  key={paper.id}
                  className={`paper-card ${selectedPapers.has(paper.id) ? 'selected' : ''}`}
                >
                  {/* カードヘッダー */}
                  <div className="card-header">
                    <input
                      type="checkbox"
                      checked={selectedPapers.has(paper.id)}
                      onChange={() => toggleSelection(paper.id)}
                      className="card-checkbox"
                    />
                    <div className="card-tags">
                      {paper.tags && paper.tags.length > 1 ? (
                        paper.tags.slice(1).map(tag => (
                          <span
                            key={tag.id}
                            className="card-tag"
                            style={{ backgroundColor: `${tag.color}20`, color: tag.color }}
                          >
                            {tag.name}
                          </span>
                        ))
                      ) : null}
                    </div>
                    {/* メインタグ（クリックでタグ設定） */}
                    <button
                      onClick={() => openTagModal(paper.id)}
                      className="card-main-tag"
                      title={paper.tags && paper.tags.length > 0 ? `タグ: ${paper.tags[0].name}` : "タグ設定"}
                      style={paper.tags && paper.tags.length > 0 ? {
                        backgroundColor: `${paper.tags[0].color}20`,
                        color: paper.tags[0].color,
                        borderColor: paper.tags[0].color
                      } : undefined}
                    >
                      {paper.tags && paper.tags.length > 0 ? paper.tags[0].name : 'タグなし'}
                    </button>
                  </div>

                  {/* カードコンテンツ */}
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

                  {/* カードアクション */}
                  <div className="card-actions">
                    <button
                      onClick={() => copyBibTeX(paper.bibtex, paper.title)}
                      className="action-btn action-copy"
                      title="BibTeXコピー"
                    >
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
                  </div>
                </article>
              ))}
            </div>

            {/* ページネーション */}
            {!isFiltered && (
              <div className="pagination-section">
                <button
                  onClick={() => setCurrentPage(Math.max(0, currentPage - 1))}
                  disabled={currentPage === 0}
                  className="pagination-btn"
                >
                  ← 前へ
                </button>
                <span className="pagination-info">
                  {currentPage + 1}ページ目
                </span>
                <button
                  onClick={() => setCurrentPage(currentPage + 1)}
                  disabled={papers.length < ITEMS_PER_PAGE}
                  className="pagination-btn"
                >
                  次へ →
                </button>
              </div>
            )}
          </>
        )}
      </main>

      {/* 登録モーダル */}
      {showRegisterModal && (
        <div className="modal-overlay" onClick={() => setShowRegisterModal(false)}>
          <div className="modal-container" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>論文登録</h2>
              <button onClick={() => setShowRegisterModal(false)} className="modal-close-btn">×</button>
            </div>
            <form onSubmit={handleRegister} className="modal-body">
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
                      onClick={() => toggleRegisterTag(tag.id)}
                      className={registerTags.includes(tag.id) ? 'tag-select-btn selected' : 'tag-select-btn'}
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
                <button type="button" onClick={() => setShowRegisterModal(false)} className="btn-secondary">
                  キャンセル
                </button>
                <button type="submit" disabled={loading || !arxivIds.trim()} className="btn-primary">
                  {loading ? '登録中...' : '登録'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* 検索モーダル */}
      {showSearchModal && (
        <div className="modal-overlay" onClick={() => setShowSearchModal(false)}>
          <div className="modal-container" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>論文検索</h2>
              <button onClick={() => setShowSearchModal(false)} className="modal-close-btn">×</button>
            </div>
            <form onSubmit={handleSearch} className="modal-body">
              <div className="form-group">
                <label htmlFor="search-query" className="form-label">検索クエリ</label>
                <input
                  id="search-query"
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="キーワードまたは自然言語"
                  disabled={loading}
                  className="form-input"
                />
              </div>
              <div className="form-group">
                <label htmlFor="search-mode" className="form-label">検索モード</label>
                <select
                  id="search-mode"
                  value={searchMode}
                  onChange={(e) => setSearchMode(e.target.value as SearchMode)}
                  disabled={loading}
                  className="form-select"
                >
                  <option value="semantic">セマンティック（意味検索）</option>
                  <option value="keyword">キーワード検索</option>
                </select>
                <p className="form-help">
                  セマンティック検索では類似度{SIMILARITY_THRESHOLD * 100}%以上の論文を表示します
                </p>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={() => setShowSearchModal(false)} className="btn-secondary">
                  キャンセル
                </button>
                <button type="submit" disabled={loading || !searchQuery} className="btn-primary">
                  {loading ? '検索中...' : '検索'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* タグ管理モーダル */}
      {showTagModal && tagModalPaperId === null && (
        <div className="modal-overlay" onClick={() => setShowTagModal(false)}>
          <div className="modal-container" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>タグ管理</h2>
              <button onClick={() => setShowTagModal(false)} className="modal-close-btn">×</button>
            </div>
            <div className="modal-body">
              {/* 新規タグ作成フォーム */}
              <form onSubmit={handleCreateTag} className="new-tag-form">
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
                  <button type="submit" disabled={loading || !newTagName} className="btn-small">
                    作成
                  </button>
                </div>
              </form>

              {/* 既存タグ一覧 */}
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
                            onClick={() => startEditTag(tag)}
                            className="tag-action-btn"
                            title="編集"
                          >
                            ✏️
                          </button>
                          <button
                            onClick={() => handleDeleteTag(tag.id, tag.name)}
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
                      <button
                        onClick={handleBulkDeleteTags}
                        className="btn-small btn-danger"
                      >
                        🗑️ 選択したタグを削除
                      </button>
                    </div>
                  )}
                </div>
              )}
            </div>
            <div className="modal-footer">
              <button onClick={() => setShowTagModal(false)} className="btn-secondary">
                閉じる
              </button>
            </div>
          </div>
        </div>
      )}

      {/* タグ編集モーダル */}
      {showTagEditModal && editingTagId !== null && (
        <div className="modal-overlay" onClick={() => setShowTagEditModal(false)}>
          <div className="modal-container" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>タグ編集</h2>
              <button onClick={() => setShowTagEditModal(false)} className="modal-close-btn">×</button>
            </div>
            <form onSubmit={handleUpdateTag} className="modal-body">
              <div className="form-group">
                <label htmlFor="edit-tag-name" className="form-label">タグ名</label>
                <input
                  id="edit-tag-name"
                  type="text"
                  value={editingTagName}
                  onChange={(e) => setEditingTagName(e.target.value)}
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
                    value={editingTagColor}
                    onChange={(e) => setEditingTagColor(e.target.value)}
                    disabled={loading}
                    className="color-input-small"
                  />
                  <span className="color-label">{editingTagColor}</span>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" onClick={cancelEditTag} className="btn-secondary">
                  キャンセル
                </button>
                <button type="submit" disabled={loading || !editingTagName} className="btn-primary">
                  {loading ? '保存中...' : '保存'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* 論文タグ設定モーダル */}
      {showTagModal && tagModalPaperId !== null && (
        <TagModal
          paperId={tagModalPaperId}
          tags={tags}
          onClose={() => { setShowTagModal(false); setTagModalPaperId(null); }}
          onSetTags={handleSetTags}
          papers={displayPapers}
          onCreateTag={async (name, color) => {
            const newTag = await createTag(name, color)
            loadTags()
            return newTag
          }}
        />
      )}
    </div>
  )
}

// タグ設定モーダルコンポーネント
function TagModal({
  paperId,
  tags,
  onClose,
  onSetTags,
  papers,
  onCreateTag
}: {
  paperId: string
  tags: Tag[]
  onClose: () => void
  onSetTags: (tagIds: number[]) => void
  papers: SearchResult[]
  onCreateTag: (name: string, color: string) => Promise<Tag>
}) {
  const [selectedTags, setSelectedTags] = useState<number[]>([])
  const [newTagName, setNewTagName] = useState('')
  const [newTagColor, setNewTagColor] = useState('#6366f1')
  const [loading, setLoading] = useState(false)

  const paper = papers.find(p => p.id === paperId)

  useEffect(() => {
    if (paper) {
      setSelectedTags(paper.tags?.map(t => t.id) || [])
    }
  }, [paper])

  const toggleTag = (tagId: number) => {
    if (selectedTags.includes(tagId)) {
      setSelectedTags(selectedTags.filter(id => id !== tagId))
    } else {
      setSelectedTags([...selectedTags, tagId])
    }
  }

  const handleCreateTag = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newTagName.trim()) return

    setLoading(true)
    try {
      const newTag = await onCreateTag(newTagName, newTagColor)
      setSelectedTags([...selectedTags, newTag.id])
      setNewTagName('')
      setNewTagColor('#6366f1')
    } catch (err) {
      console.error('タグ作成エラー:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleSave = () => {
    onSetTags(selectedTags)
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-container" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>タグ設定</h2>
          <button onClick={onClose} className="modal-close-btn">×</button>
        </div>
        <div className="modal-body">
          <p className="paper-title-preview">{paper?.title}</p>

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

          <form onSubmit={handleCreateTag} className="new-tag-section">
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
              <button type="submit" disabled={loading || !newTagName} className="btn-small">
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

export default App
