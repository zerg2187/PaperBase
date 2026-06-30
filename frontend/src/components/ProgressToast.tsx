type ProgressToastProps = {
  current: number
  total: number
}

export function ProgressToast({ current, total }: ProgressToastProps) {
  const percentage = total > 0 ? (current / total) * 100 : 0

  return (
    <div className="progress-toast">
      <div className="progress-toast-header">
        <span className="progress-toast-title">論文登録中...</span>
        <span className="progress-toast-count">{current}/{total}</span>
      </div>
      <div className="progress-bar">
        <div
          className="progress-bar-fill"
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  )
}
