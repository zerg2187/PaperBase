import { useState, useCallback } from 'react'

export function useToast() {
  const [successMessage, setSuccessMessage] = useState<string | null>(null)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const showSuccess = useCallback((message: string, duration = 3000) => {
    setSuccessMessage(message)
    setTimeout(() => setSuccessMessage(null), duration)
  }, [])

  const showError = useCallback((message: string, duration = 3000) => {
    setErrorMessage(message)
    setTimeout(() => setErrorMessage(null), duration)
  }, [])

  const clearSuccess = useCallback(() => setSuccessMessage(null), [])
  const clearError = useCallback(() => setErrorMessage(null), [])

  return {
    successMessage,
    errorMessage,
    showSuccess,
    showError,
    clearSuccess,
    clearError,
  }
}
