import '@testing-library/jest-dom'
import { cleanup } from '@testing-library/react'
import { afterEach, beforeAll, afterAll } from 'vitest'
import { setupServer } from 'msw/node'
import { handlers } from './mocks/handlers'

// MSW サーバーのセットアップ
export const mockServer = setupServer(...handlers)

// テスト開始前にMSWサーバーを起動
beforeAll(() => {
  mockServer.listen({ onUnhandledRequest: 'bypass' })
})

// テストごとにハンドラーをリセット
afterEach(() => {
  cleanup()
  mockServer.resetHandlers()
})

// 全テスト終了後にMSWサーバーを停止
afterAll(() => {
  mockServer.close()
})
