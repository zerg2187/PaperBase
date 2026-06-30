// =============================================================================
// ランダムタグ色ユーティリティ
// =============================================================================

export const tagColors = [
  '#ef4444',
  '#f97316',
  '#f59e0b',
  '#84cc16',
  '#22c55e',
  '#14b8a6',
  '#06b6d4',
  '#3b82f6',
  '#6366f1',
  '#8b5cf6',
  '#a855f7',
  '#d946ef',
  '#ec4899',
  '#f43f5e',
]

export function randomColor(): string {
  return tagColors[Math.floor(Math.random() * tagColors.length)]
}
