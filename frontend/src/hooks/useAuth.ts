import { useState, useEffect, useCallback } from 'react'
import { getGuestStatus, loginAdmin, logoutAdmin, setAdminToken } from '../api'

export function useAuth() {
  const [authRole, setAuthRole] = useState<'admin' | 'guest' | null>(null)
  const [sessionId, setSessionId] = useState('')
  const [guestRemainingCount, setGuestRemainingCount] = useState<number | null>(null)

  const [showLoginModal, setShowLoginModal] = useState(false)
  const [loginToken, setLoginToken] = useState('')
  const [loginError, setLoginError] = useState<string | null>(null)
  const [loginLoading, setLoginLoading] = useState(false)

  const loadAuthStatus = useCallback(async () => {
    try {
      const status = await getGuestStatus()
      setAuthRole(status.role as 'admin' | 'guest')
      setSessionId(status.session_id)
      if (status.role === 'guest' && status.remaining_paper_count !== undefined) {
        setGuestRemainingCount(status.remaining_paper_count)
      } else {
        setGuestRemainingCount(null)
      }
    } catch (err) {
      console.error('認証状態取得エラー:', err)
    }
  }, [])

  useEffect(() => {
    loadAuthStatus()
  }, [loadAuthStatus])

  const login = useCallback(async (token: string): Promise<{ success: boolean; error?: string }> => {
    setLoginError(null)
    setLoginLoading(true)
    try {
      await loginAdmin(token)
      return { success: true }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'ログインに失敗しました'
      setLoginError(message)
      return { success: false, error: message }
    } finally {
      setLoginLoading(false)
    }
  }, [])

  const logout = useCallback(async (): Promise<{ success: boolean; error?: string }> => {
    try {
      await logoutAdmin()
      setAdminToken(null)
      window.location.reload()
      return { success: true }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'ログアウトに失敗しました'
      return { success: false, error: message }
    }
  }, [])

  return {
    authRole,
    sessionId,
    guestRemainingCount,
    showLoginModal,
    setShowLoginModal,
    loginToken,
    setLoginToken,
    loginError,
    setLoginError,
    loginLoading,
    login,
    logout,
    loadAuthStatus,
  }
}
