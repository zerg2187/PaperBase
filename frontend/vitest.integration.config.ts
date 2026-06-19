/// <reference types="vitest" />
import react from '@vitejs/plugin-react'
import path from 'path'

export default {
  plugins: [react()],
  test: {
    globals: true,
    environment: 'node', // Node環境で実行
    setupFiles: [], // MSW setupを使用しない
    include: ['src/**/*.integration.test.{ts,tsx}'],
    exclude: ['**/node_modules/**', '**/dist/**'],
    reporter: ['verbose'], // 詳細なログを出力
    env: {
      VITE_API_BASE_URL: 'http://localhost:8080',
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
}
