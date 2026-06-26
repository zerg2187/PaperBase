import { useState, useCallback } from 'react'
import { registerPaper, setPaperTags } from '../api'

export function usePaperRegistration() {
  const [registerLoading, setRegisterLoading] = useState(false)
  const [registerProgress, setRegisterProgress] = useState<{ current: number; total: number } | null>(null)

  const register = useCallback(async (ids: string[], tagIds: number[]) => {
    setRegisterLoading(true)
    setRegisterProgress(null)

    const results = { success: 0, failed: 0, errors: [] as string[] }

    for (let i = 0; i < ids.length; i++) {
      const id = ids[i]
      setRegisterProgress({ current: i + 1, total: ids.length })
      try {
        await registerPaper(id)
        if (tagIds.length > 0) {
          await setPaperTags(id, tagIds)
        }
        results.success++
      } catch (err) {
        results.failed++
        results.errors.push(`${id}: ${err instanceof Error ? err.message : '登録失敗'}`)
      }
    }

    setRegisterLoading(false)
    setRegisterProgress(null)
    return results
  }, [])

  return {
    registerLoading,
    registerProgress,
    register,
  }
}
