import { useState, useCallback } from 'react'
import { getPapersPaginated, searchPapers, deletePaper, deletePapers } from '../api'
import type { SearchResult, SearchMode } from '../types'

const SIMILARITY_THRESHOLD = 0.30
const FETCH_ALL_LIMIT = 10000

export function usePapers(authRole: 'admin' | 'guest' | null) {
  const [papers, setPapers] = useState<SearchResult[]>([])
  const [filteredPapers, setFilteredPapers] = useState<SearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedTagFilter, setSelectedTagFilter] = useState<number | null>(null)
  const [isFiltered, setIsFiltered] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchMode, setSearchMode] = useState<SearchMode>('semantic')

  const loadPapers = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getPapersPaginated(0, FETCH_ALL_LIMIT, selectedTagFilter)
      setPapers(data)
    } finally {
      setLoading(false)
    }
  }, [selectedTagFilter])

  const search = useCallback(async (query: string, mode: SearchMode) => {
    setLoading(true)
    try {
      const data = await searchPapers(query, mode)
      // しきい値は admin のみ。ゲストは最大5件の小コーパスで全滅しやすいため全件表示する
      const filtered = data.filter(p => {
        if (authRole === 'admin' && mode === 'semantic' && p.similarity !== undefined) {
          return p.similarity >= SIMILARITY_THRESHOLD
        }
        return true
      })
      const sorted = [...filtered].sort((a, b) => {
        return (b.similarity ?? 0) - (a.similarity ?? 0)
      })
      setFilteredPapers(sorted)
      setIsFiltered(true)
      // 検索とタグ絞り込みは排他。タグが残ると displayPapers が検索結果を表示しない
      setSelectedTagFilter(null)
      setSearchQuery(query)
      setSearchMode(mode)
      return sorted
    } finally {
      setLoading(false)
    }
  }, [authRole])

  const clearSearch = useCallback(() => {
    setFilteredPapers([])
    setIsFiltered(false)
    setSearchQuery('')
  }, [])

  const clearTagFilter = useCallback(() => {
    setSelectedTagFilter(null)
    setIsFiltered(false)
  }, [])

  const handleDeletePaper = useCallback(async (id: string) => {
    await deletePaper(id)
    setPapers(prev => prev.filter(p => p.id !== id))
    setFilteredPapers(prev => prev.filter(p => p.id !== id))
  }, [])

  const handleBulkDelete = useCallback(async (ids: string[]) => {
    await deletePapers(ids)
    setPapers(prev => prev.filter(p => !ids.includes(p.id)))
    setFilteredPapers(prev => prev.filter(p => !ids.includes(p.id)))
  }, [])

  const displayPapers = (isFiltered && selectedTagFilter === null) ? filteredPapers : papers

  return {
    papers,
    filteredPapers,
    loading,
    selectedTagFilter,
    setSelectedTagFilter,
    isFiltered,
    searchQuery,
    searchMode,
    loadPapers,
    search,
    clearSearch,
    clearTagFilter,
    deletePaper: handleDeletePaper,
    bulkDelete: handleBulkDelete,
    displayPapers,
  }
}
