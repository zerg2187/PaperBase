import { useEffect } from 'react'

type ToastProps = {
  message: string
  type: 'success' | 'error'
  onClose: () => void
}

export function Toast({ message, type, onClose }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(onClose, 4000)
    return () => clearTimeout(timer)
  }, [onClose])

  return (
    <div className={`toast toast-${type}`} onClick={onClose}>
      <span className="toast-icon">{type === 'success' ? '✓' : '⚠'}</span>
      <span className="toast-message">{message}</span>
      <button
        className="toast-close"
        onClick={(e) => { e.stopPropagation(); onClose() }}
        aria-label="閉じる"
      >
        ×
      </button>
      <div className="toast-progress" />
    </div>
  )
}
