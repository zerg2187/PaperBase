/**
 * App コンポーネントのテスト
 *
 * MSW（Mock Service Worker）を使用したモックテスト
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from './App'

describe('App Component', () => {
  beforeEach(() => {
    // MSWサーバーはsetup.tsでグローバルに設定済み
  })

  it('should render the app title', async () => {
    render(<App />)

    await waitFor(() => {
      expect(screen.getByText('Paperbase')).toBeInTheDocument()
    })
  })

  it('should display papers', async () => {
    const { container } = render(<App />)

    await waitFor(() => {
      const papers = container.querySelectorAll('.paper-card')
      expect(papers.length).toBeGreaterThan(0)
    })
  })

  it('should have navigation buttons', async () => {
    render(<App />)

    await waitFor(() => {
      expect(screen.getByText('検索')).toBeInTheDocument()
      expect(screen.getByText('登録')).toBeInTheDocument()
      expect(screen.getByText('タグ管理')).toBeInTheDocument()
    })
  })

  it('should open search modal when clicking search button', async () => {
    const user = userEvent.setup()
    render(<App />)

    const searchButton = await screen.findByText('検索')
    await user.click(searchButton)

    await waitFor(() => {
      expect(screen.getByText('検索クエリ')).toBeInTheDocument()
    })
  })

  it('should open register modal when clicking register button', async () => {
    const user = userEvent.setup()
    render(<App />)

    const registerButton = await screen.findByText('登録')
    await user.click(registerButton)

    await waitFor(() => {
      expect(screen.getByText('arXiv ID')).toBeInTheDocument()
    })
  })

  it('should display tags', async () => {
    render(<App />)

    await waitFor(() => {
      expect(screen.getByText('LLM')).toBeInTheDocument()
      expect(screen.getByText('Vision')).toBeInTheDocument()
    })
  })

  it('should show tag filter section', async () => {
    render(<App />)

    await waitFor(() => {
      expect(screen.getByText('タグで絞り込み:')).toBeInTheDocument()
      expect(screen.getByText('すべて')).toBeInTheDocument()
    })
  })

  it('should open tag edit modal when clicking edit button in tag management', async () => {
    const user = userEvent.setup()
    render(<App />)

    const tagManageButton = await screen.findByText('タグ管理')
    await user.click(tagManageButton)

    const tagEditButton = await screen.findByTitle('編集')
    await user.click(tagEditButton)

    await waitFor(() => {
      expect(screen.getByText('タグ編集')).toBeInTheDocument()
      expect(screen.getByLabelText('タグ名')).toBeInTheDocument()
      expect(screen.getByLabelText('色')).toBeInTheDocument()
    })
  })

