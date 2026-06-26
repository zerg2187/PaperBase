import { useState, useEffect } from 'react'
import './App.css'
import { setPaperTags } from './api'
import type { SearchResult, SearchMode } from './types'

import { Header } from './components/Header'
import { FilterBar } from './components/FilterBar'
import { TagFilter } from './components/TagFilter'
import { PaperList } from './components/PaperList'
import { Toast } from './components/Toast'

import { RegisterModal } from './components/modals/RegisterModal'
import { SearchModal } from './components/modals/SearchModal'
import { TagManagerModal } from './components/modals/TagManagerModal'
import { TagSelectModal } from './components/modals/TagSelectModal'
import { LoginModal } from './components/modals/LoginModal'

import { useAuth } from './hooks/useAuth'
import { useToast } from './hooks/useToast'
import { useSelection } from './hooks/useSelection'
import { useTags } from './hooks/useTags'
import { usePapers } from './hooks/usePapers'
import { usePaperRegistration } from './hooks/usePaperRegistration'

const ITEMS_PER_PAGE = 10

function App() {
  // Hooks
  const {
    papers,
    filteredPapers,
    loading: papersLoading,
    currentPage,
    setCurrentPage,
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
  } = usePapers()

  const { tags, loading: tagsLoading, loadTags, createTag: createTagHook, updateTag: updateTagHook, deleteTag: deleteTagHook, bulkDeleteTags } = useTags()

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

  const { successMessage, errorMessage, showSuccess, showError, clearSuccess, clearError } = useToast()
  const { selected: selectedPapers, toggle: toggleSelection, selectAll: selectAllPapers, clear: clearSelection } = useSelection<string>()
  const { registerLoading, registerProgress, register: registerPapers } = usePaperRegistration()

  // Modal states
  const [showRegisterModal, setShowRegisterModal] = useState(false)
  const [showSearchModal, setShowSearchModal] = useState(false)
  const [showTagManagerModal, setShowTagManagerModal] = useState(false)
  const [tagSelectPaper, setTagSelectPaper] = useState<SearchResult | null>(null)

  // Initial load
  useEffect(() => {
    loadPapers()
    loadTags()
  }, [loadPapers, loadTags])

  // Selection state derived from display papers
  const displayCount = displayPapers.length
  const hasNextPage = papers.length >= ITEMS_PER_PAGE

  // Auth handlers
  const handleLogin = async (token: string) => {
    const result = await login(token)
    if (result.success) {
      setShowLoginModal(false)
      showSuccess('管理者としてログインしました')
    }
  }

  const handleLogout = async () => {
    const result = await logout()
    if (result.success) {
      showSuccess('ログアウトしました')
    } else {
      showError(result.error)
    }
  }

  // Paper handlers
  const handleDeletePaper = async (paper: SearchResult) => {
    if (!confirm(`「${paper.title.slice(0, 30)}...」を削除しますか？`)) return

    try {
      await deletePaperFromList(paper.id)
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
      clearSelection()
      selectAllPapers(results.map(p => p.id))
      showSuccess(`${results.length}件の論文が見つかりました`)
    } catch (err) {
      showError(err instanceof Error ? err.message : '検索に失敗しました')
    }
  }

  const clearFilter = () => {
    clearSearch()
    clearTagFilter()
    clearSelection()
  }

  const handleRegister = async (ids: string[], tagIds: number[]) => {
    const results = await registerPapers(ids, tagIds)
    setShowRegisterModal(false)

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
    if (selectedPapers.size === displayCount) {
      clearSelection()
    } else {
      selectAllPapers(displayPapers.map(p => p.id))
    }
  }

  const copyBibTeX = (bibtex: string, title: string) => {
    navigator.clipboard.writeText(bibtex)
    showSuccess(`「${title.slice(0, 20)}...」のBibTeXをコピーしました`)
  }

  const exportBibTeX = () => {
    const selectedPapersData = displayPapers.filter(p => selectedPapers.has(p.id))

    if (selectedPapersData.length === 0) {
      showError('論文が選択されていません')
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

    showSuccess(`${selectedPapersData.length}件のBibTeXをエクスポートしました`)
  }

  const handleBulkDelete = async () => {
    const selectedIds = Array.from(selectedPapers).filter(id =>
      displayPapers.some(p => p.id === id)
    )

    if (selectedIds.length === 0) {
      showError('論文が選択されていません')
      return
    }

    if (!confirm(`${selectedIds.length}件の論文を削除しますか？`)) return

    try {
      await bulkDelete(selectedIds)
      showSuccess(`${selectedIds.length}件の論文を削除しました`)
      clearSelection()
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
    setSelectedTagFilter(tagId)
    setCurrentPage(0)
  }

  return (
    <div className="app">
      <Header
        paperCount={displayCount}
        selectedCount={selectedPapers.size}
        authRole={authRole}
        guestRemainingCount={guestRemainingCount}
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

      <FilterBar
        isFiltered={isFiltered}
        selectedTagFilter={selectedTagFilter}
        tags={tags}
        count={selectedTagFilter !== null ? papers.length : filteredPapers.length}
        onClear={clearFilter}
      />

      <TagFilter
        tags={tags}
        selectedTagFilter={selectedTagFilter}
        onSelect={handleTagFilterSelect}
      />

      <main className="main-content">
        <PaperList
          papers={displayPapers}
          selectedPapers={selectedPapers}
          authRole={authRole}
          loading={papersLoading || tagsLoading}
          showPagination={!isFiltered || selectedTagFilter !== null}
          currentPage={currentPage}
          hasNext={hasNextPage}
          onToggle={toggleSelection}
          onSelectAll={toggleSelectAll}
          onPrevPage={() => setCurrentPage(Math.max(0, currentPage - 1))}
          onNextPage={() => setCurrentPage(currentPage + 1)}
          onTagClick={openTagSelectModal}
          onDelete={handleDeletePaper}
          onCopyBibTeX={copyBibTeX}
          onRegisterClick={() => setShowRegisterModal(true)}
        />
      </main>

      <RegisterModal
        isOpen={showRegisterModal}
        tags={tags}
        loading={registerLoading}
        progress={registerProgress}
        onClose={() => setShowRegisterModal(false)}
        onRegister={handleRegister}
      />

      <SearchModal
        isOpen={showSearchModal}
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
