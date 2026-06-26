import { useState, useCallback } from 'react'
import { getAllTags, createTag, updateTag, deleteTag } from '../api'
import type { Tag } from '../types'

export function useTags() {
  const [tags, setTags] = useState<Tag[]>([])
  const [loading, setLoading] = useState(false)

  const loadTags = useCallback(async () => {
    try {
      const data = await getAllTags()
      setTags(data)
    } catch (err) {
      console.error('タグ読み込みエラー:', err)
    }
  }, [])

  const handleCreateTag = useCallback(async (name: string, color: string) => {
    setLoading(true)
    try {
      const tag = await createTag(name, color)
      setTags(prev => [...prev, tag])
      return tag
    } finally {
      setLoading(false)
    }
  }, [])

  const handleUpdateTag = useCallback(async (id: number, name: string, color: string) => {
    setLoading(true)
    try {
      await updateTag(id, name, color)
      setTags(prev => prev.map(t => (t.id === id ? { ...t, name, color } : t)))
    } finally {
      setLoading(false)
    }
  }, [])

  const handleDeleteTag = useCallback(async (id: number) => {
    setLoading(true)
    try {
      await deleteTag(id)
      setTags(prev => prev.filter(t => t.id !== id))
    } finally {
      setLoading(false)
    }
  }, [])

  const bulkDeleteTags = useCallback(async (ids: number[]) => {
    setLoading(true)
    try {
      for (const id of ids) {
        await deleteTag(id)
      }
      setTags(prev => prev.filter(t => !ids.includes(t.id)))
    } finally {
      setLoading(false)
    }
  }, [])

  return {
    tags,
    loading,
    loadTags,
    createTag: handleCreateTag,
    updateTag: handleUpdateTag,
    deleteTag: handleDeleteTag,
    bulkDeleteTags,
    setTags,
  }
}
