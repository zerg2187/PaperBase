import { useState, useCallback } from 'react'

export function useSelection<T>() {
  const [selected, setSelected] = useState<Set<T>>(new Set())

  const toggle = useCallback((id: T) => {
    setSelected(prev => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const selectAll = useCallback((ids: T[]) => {
    setSelected(new Set(ids))
  }, [])

  const clear = useCallback(() => {
    setSelected(new Set())
  }, [])

  const isAllSelected = useCallback(
    (ids: T[]) => ids.length > 0 && ids.every(id => selected.has(id)),
    [selected]
  )

  return {
    selected,
    toggle,
    selectAll,
    clear,
    setSelected,
    isAllSelected,
  }
}
