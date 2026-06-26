import { useState, useCallback } from 'react'
import { getPapersPaginated, searchPapers, deletePaper, deletePapers } from '../api'
import type { SearchResult, SearchMode } from '../types'

const ITEMS_PER_PAGE = 10
const SIMILARITY_THRESHOLD = 0.30

export function usePapers() {
  const [papers, setPapers] = useState<SearchResult[]>([])
  const [filteredPapers, setFilteredPapers] = useState<SearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const [currentPage, setCurrentPage] = useState(0)
  const [selectedTagFilter, setSelectedTagFilter] = useState<number | null>(null)
  const [isFiltered, setIsFiltered] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchMode, setSearchMode] = useState<SearchMode>('semantic')

  const loadPapers = useCallback(async () => {
    setLoading(true)
    try {
      const offset = currentPage * ITEMS_PER_PAGE
      const data = await getPapersPaginated(offset, ITEMS_PER_PAGE, selectedTagFilter)
      setPapers(data)
    } finally {
      setLoading(false)
    }
  }, [currentPage, selectedTagFilter])

  const search = useCallback(async (query: string, mode: SearchMode) => {
    setLoading(true)
    try {
      const data = await searchPapers(query, mode)
      const filtered = data.filter(p => {
        if (mode === 'semantic' && p.similarity !== undefined) {
          return p.similarity >= SIMILARITY_THRESHOLD
        }
        return true
      })
      const sorted = [...filtered].sort((a, b) => {
        return (b.similarity ?? 0) - (a.similarity ?? 0)
      })
      setFilteredPapers(sorted)
      setIsFiltered(true)
      setSearchQuery(query)
      setSearchMode(mode)
      return sorted
    } finally {
      setLoading(false)
    }
  }, [])

  const clearSearch = useCallback(() => {
    setFilteredPapers([])
    setIsFiltered(false)
    setSearchQuery('')
  }, [])

  const clearTagFilter = useCallback(() => {
    setSelectedTagFilter(null)
    setCurrentPage(0)
    setIsFiltered(false)
  }, [])

  const handleDeletePaper = useCallback(async (id: string) => {
    await deletePaper(id)
    setPapers(prev => prev.filter(p => p.id !== id))
    setFilteredPapers(prev => prev.filter(p => p.id !== id))
  }, [])

  const handleBulkDelete = useCallback(async (ids: string[]) => {
    await deletePapers(ids)
    setPapers(prev => {
      const remaining = prev.filter(p => !ids.includes(p.id))
      if (remaining.length === 0 && currentPage > 0) {
        setCurrentPage(p => p - 1)
      }
      return remaining
    })
    setFilteredPapers(prev => prev.filter(p => !ids.includes(p.id)))
  }, [currentPage])

  const displayPapers = (isFiltered && selectedTagFilter === null) ? filteredPapers : papers

  return {
    papers,
    filteredPapers,
    loading,
    currentPage,
    setCurrentPage,
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
