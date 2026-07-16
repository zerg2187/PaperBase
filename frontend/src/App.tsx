import { useState, useEffect } from 'react'
import './App.css'
import { setPaperTags } from './api'
import type { SearchResult, SearchMode } from './types'

import { Header } from './components/Header'
import { FilterBar } from './components/FilterBar'
import { TagFilter } from './components/TagFilter'
import { PaperList } from './components/PaperList'
import { DetailPanel } from './components/DetailPanel'
import { CartPanel } from './components/CartPanel'
import { ProgressToast } from './components/ProgressToast'
import { Toast } from './components/Toast'

import { RegisterModal } from './components/modals/RegisterModal'
import { SearchModal } from './components/modals/SearchModal'
import { TagManagerModal } from './components/modals/TagManagerModal'
import { TagSelectModal } from './components/modals/TagSelectModal'
import { LoginModal } from './components/modals/LoginModal'

import { useAuth } from './hooks/useAuth'
import { useToast } from './hooks/useToast'
import { useTags } from './hooks/useTags'
import { usePapers } from './hooks/usePapers'
import { usePaperRegistration } from './hooks/usePaperRegistration'
import { useCart, clearCartStorage } from './hooks/useCart'

function App() {
  // Hooks
  const {
    authRole,
    guestRemainingCount,
    showLoginModal,
    setShowLoginModal,
    loginError,
    loginLoading,
    login,
    logout,
    loadAuthStatus,
  } = useAuth()

  const {
    papers,
    filteredPapers,
    loading: papersLoading,
    selectedTagFilter,
    setSelectedTagFilter,
    isFiltered,
    loadPapers,
    search,
    clearSearch,
    clearTagFilter,
    deletePaper: deletePaperFromList,
    bulkDelete,
    displayPapers,
  } = usePapers(authRole)

  const { tags, loading: tagsLoading, loadTags, createTag: createTagHook, updateTag: updateTagHook, deleteTag: deleteTagHook, bulkDeleteTags } = useTags()

  const { successMessage, errorMessage, showSuccess, showError, clearSuccess, clearError } = useToast()
  const { registerLoading, registerProgress, register: registerPapers } = usePaperRegistration()
  const { cart, addToCart, removeFromCart, clearCart, isInCart } = useCart()

  // Modal states
  const [showRegisterModal, setShowRegisterModal] = useState(false)
  const [showSearchModal, setShowSearchModal] = useState(false)
  const [showTagManagerModal, setShowTagManagerModal] = useState(false)
  const [tagSelectPaper, setTagSelectPaper] = useState<SearchResult | null>(null)
  const [showCartPanel, setShowCartPanel] = useState(false)

  // Detail panel state
  const [activePaper, setActivePaper] = useState<SearchResult | null>(null)
  // クリックしたカードの真横に詳細を表示するためのオフセット(main-content 基準)
  const [activeCardTop, setActiveCardTop] = useState(0)

  const handleItemClick = (paper: SearchResult, offsetTop: number) => {
    setActivePaper(paper)
    setActiveCardTop(offsetTop)
  }

  // Cart checkbox handler
  const handleToggleCart = (paperId: string) => {
    const paper = displayPapers.find(p => p.id === paperId)
    if (!paper) return

    if (isInCart(paperId)) {
      removeFromCart(paperId)
    } else {
      addToCart(paper)
    }
  }

  // Initial load
  // authRole が確定(admin/guest)してから取得する。null(認証確認中)のまま fetch すると
  // サーバ側のセッション/ロール確定前に走り、ログイン直後に一覧が0件になることがある。
  useEffect(() => {
    if (authRole === null) return
    loadPapers()
    if (authRole === 'admin') {
      loadTags()
    }
  }, [loadPapers, loadTags, authRole])

  // Selection state derived from display papers
  const displayCount = displayPapers.length

  // Auth handlers
  const handleLogin = async (token: string) => {
    const result = await login(token)
    if (result.success) {
      clearCartStorage()
      window.location.reload()
    }
  }

  const handleLogout = async () => {
    const result = await logout()
    if (result.success) {
      showSuccess('ログアウトしました')
    } else {
      showError(result.error || 'ログアウトに失敗しました')
    }
  }

  // Paper handlers
  const handleDeletePaper = async (paper: SearchResult) => {
    if (!confirm(`「${paper.title.slice(0, 30)}...」を削除しますか？`)) return

    try {
      await deletePaperFromList(paper.id)
      removeFromCart(paper.id)
      if (activePaper?.id === paper.id) {
        setActivePaper(null)
      }
      showSuccess('論文を削除しました')
      await loadPapers()
      await loadAuthStatus()
    } catch (err) {
      showError(err instanceof Error ? err.message : '削除に失敗しました')
    }
  }

  const handleSearch = async (query: string, mode: SearchMode) => {
    try {
      const results = await search(query, mode)
      showSuccess(`${results.length}件の論文が見つかりました`)
    } catch (err) {
      showError(err instanceof Error ? err.message : '検索に失敗しました')
    }
  }

  const clearFilter = () => {
    clearSearch()
    clearTagFilter()
  }

  const handleRegister = async (ids: string[], tagIds: number[]) => {
    setShowRegisterModal(false)
    const results = await registerPapers(ids, tagIds)

    if (results.success > 0) {
      showSuccess(`${results.success}件の論文を登録しました`)
      await loadPapers()
      await loadAuthStatus()
    }

    if (results.failed > 0) {
      setTimeout(
        () => showError(`${results.failed}件失敗: ${results.errors.join(', ')}`),
        results.success > 0 ? 3000 : 0
      )
    }
  }

  const toggleSelectAll = () => {
    // Use current cart state directly to avoid stale closure
    const currentCartIds = new Set(cart.map(c => c.id))
    const allInCart = displayPapers.length > 0 && displayPapers.every(p => currentCartIds.has(p.id))

    if (allInCart) {
      clearCart()
    } else {
      displayPapers.forEach(p => {
        if (!currentCartIds.has(p.id)) {
          addToCart(p)
        }
      })
    }
  }

  const copyBibTeX = (bibtex: string, title: string) => {
    navigator.clipboard.writeText(bibtex)
    showSuccess(`「${title.slice(0, 20)}...」のBibTeXをコピーしました`)
  }

  const exportBibTeX = () => {
    if (cart.length === 0) {
      showError('カートが空です')
      return
    }

    const bibTeXContent = cart.map(p => p.bibtex).join('\n\n')
    const blob = new Blob([bibTeXContent], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'papers.bib'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)

    showSuccess(`${cart.length}件のBibTeXをエクスポートしました`)
    setShowCartPanel(false)
  }

  const exportPresentation = () => {
    if (cart.length === 0) {
      showError('カートが空です')
      return
    }

    const lastName = (name: string) => name.trim().split(/\s+/).pop() ?? name
    const content = cart.map(p => {
      const authors = p.authors.length > 1
        ? `${lastName(p.authors[0])} et al.`
        : (p.authors[0] ?? '')
      return `${p.title}\n${authors} (${p.year})`
    }).join('\n\n')

    navigator.clipboard.writeText(content)
    showSuccess(`${cart.length}件のプレゼンテーション用テキストをコピーしました`)
    setShowCartPanel(false)
  }

  const handleBulkDelete = async () => {
    const cartIds = cart.map(c => c.id)

    if (cartIds.length === 0) {
      showError('カートが空です')
      return
    }

    if (!confirm(`${cartIds.length}件の論文を削除しますか？`)) return

    try {
      await bulkDelete(cartIds)
      showSuccess(`${cartIds.length}件の論文を削除しました`)
      clearCart()
      if (activePaper && cartIds.includes(activePaper.id)) {
        setActivePaper(null)
      }
      await loadPapers()
      await loadAuthStatus()
    } catch (err) {
      showError(err instanceof Error ? err.message : '削除に失敗しました')
    }
  }

  // Tag handlers
  const handleCreateTag = async (name: string, color: string) => {
    await createTagHook(name, color)
    showSuccess(`タグ「${name}」を作成しました`)
    await loadTags()
  }

  const handleDeleteTag = async (id: number, name: string) => {
    await deleteTagHook(id)
    showSuccess(`タグ「${name}」を削除しました`)

    if (selectedTagFilter === id) {
      clearTagFilter()
    }

    await loadTags()
    await loadPapers()
  }

  const handleUpdateTag = async (id: number, name: string, color: string) => {
    await updateTagHook(id, name, color)
    showSuccess(`タグ「${name}」を更新しました`)
    await loadTags()
  }

  const handleBulkDeleteTags = async (ids: number[]) => {
    await bulkDeleteTags(ids)
    showSuccess(`${ids.length}件のタグを削除しました`)

    if (selectedTagFilter !== null && ids.includes(selectedTagFilter)) {
      clearTagFilter()
    }

    await loadTags()
    await loadPapers()
  }

  const handleSetTags = async (paperId: string, tagIds: number[]) => {
    try {
      await setPaperTags(paperId, tagIds)
      showSuccess('タグを更新しました')
      setTagSelectPaper(null)
      await loadPapers()
    } catch (err) {
      showError(err instanceof Error ? err.message : 'タグ更新に失敗しました')
    }
  }

  const openTagSelectModal = (paper: SearchResult) => {
    setTagSelectPaper(paper)
  }

  const closeTagSelectModal = () => {
    setTagSelectPaper(null)
  }

  const handleTagFilterSelect = (tagId: number | null) => {
    // タグと検索は排他。検索中にタグを選択/解除すると isFiltered が残り、
    // 「すべて」に戻したとき displayPapers が古い検索結果になって欠けるため、検索状態を解除する。
    clearSearch()
    setSelectedTagFilter(tagId)
  }

  return (
    <div className="app">
      <Header
        paperCount={displayCount}
        selectedCount={cart.length}
        authRole={authRole}
        guestRemainingCount={guestRemainingCount}
        registerLoading={registerLoading}
        onLoginClick={() => setShowLoginModal(true)}
        onLogout={handleLogout}
        onSearchClick={() => setShowSearchModal(true)}
        onRegisterClick={() => setShowRegisterModal(true)}
        onTagManageClick={() => setShowTagManagerModal(true)}
        onExportBibTeX={exportBibTeX}
        onBulkDelete={handleBulkDelete}
      />

      {successMessage && (
        <Toast key={`success-${successMessage}`} message={successMessage} type="success" onClose={clearSuccess} />
      )}
      {errorMessage && (
        <Toast key={`error-${errorMessage}`} message={errorMessage} type="error" onClose={clearError} />
      )}

      {registerLoading && registerProgress && (
        <ProgressToast
          current={registerProgress.current}
          total={registerProgress.total}
        />
      )}

      <FilterBar
        isFiltered={isFiltered}
        selectedTagFilter={selectedTagFilter}
        tags={tags}
        count={selectedTagFilter !== null ? papers.length : filteredPapers.length}
        onClear={clearFilter}
      />

      {authRole === 'admin' && (
        <TagFilter
          tags={tags}
          selectedTagFilter={selectedTagFilter}
          onSelect={handleTagFilterSelect}
        />
      )}

      <main className="main-content">
        <div className="paper-list-zone">
          <PaperList
            papers={displayPapers}
            selectedPapers={new Set(cart.map(c => c.id))}
            activePaperId={activePaper?.id ?? null}
            authRole={authRole}
            loading={papersLoading || tagsLoading}
            isInCart={isInCart}
            onToggle={handleToggleCart}
            onSelectAll={toggleSelectAll}
            onItemClick={handleItemClick}
            onRegisterClick={() => setShowRegisterModal(true)}
          />
        </div>

        {activePaper && (
          <div className="detail-panel-zone" style={{ marginTop: activeCardTop }}>
            <DetailPanel
              paper={activePaper}
              authRole={authRole}
              onClose={() => setActivePaper(null)}
              onTagClick={() => openTagSelectModal(activePaper)}
              onDelete={() => handleDeletePaper(activePaper)}
              onCopyBibTeX={() => copyBibTeX(activePaper.bibtex, activePaper.title)}
            />
          </div>
        )}
      </main>

      <CartPanel
        cart={cart}
        onRemove={removeFromCart}
        onClear={clearCart}
        onExport={exportBibTeX}
        onExportPresentation={exportPresentation}
        isOpen={showCartPanel}
        onToggle={() => setShowCartPanel(!showCartPanel)}
      />

      <RegisterModal
        isOpen={showRegisterModal}
        authRole={authRole}
        tags={tags}
        loading={registerLoading}
        progress={registerProgress}
        onClose={() => setShowRegisterModal(false)}
        onRegister={handleRegister}
        onCreateTag={async (name, color) => {
          const tag = await createTagHook(name, color)
          showSuccess(`タグ「${name}」を作成しました`)
          await loadTags()
          return tag
        }}
      />

      <SearchModal
        isOpen={showSearchModal}
        authRole={authRole}
        loading={papersLoading}
        onClose={() => setShowSearchModal(false)}
        onSearch={handleSearch}
      />

      <TagManagerModal
        isOpen={showTagManagerModal}
        tags={tags}
        loading={tagsLoading}
        onClose={() => setShowTagManagerModal(false)}
        onCreateTag={handleCreateTag}
        onUpdateTag={handleUpdateTag}
        onDeleteTag={handleDeleteTag}
        onBulkDeleteTags={handleBulkDeleteTags}
      />

      {tagSelectPaper && (
        <TagSelectModal
          paper={tagSelectPaper}
          tags={tags}
          onClose={closeTagSelectModal}
          onSetTags={handleSetTags}
          onCreateTag={async (name, color) => {
            const tag = await createTagHook(name, color)
            await loadTags()
            return tag
          }}
        />
      )}

      <LoginModal
        isOpen={showLoginModal}
        error={loginError}
        loading={loginLoading}
        onClose={() => setShowLoginModal(false)}
        onLogin={handleLogin}
      />
    </div>
  )
}

export default App
