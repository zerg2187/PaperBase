/**
 * API クライアントの統合テスト
 *
 * 実行方法: npm run test:integration
 */

import { describe, it, expect, beforeAll } from 'vitest'
import {
  registerPaper,
  searchPapers,
  healthCheck,
  getPapersPaginated,
  getAllTags,
  createTag,
  deletePaper,
  deletePapers,
} from './api'

// グローバルスコープで宣言
let backendAvailable = false
let testPaperId: string | null = null
let testTagId: number | null = null

beforeAll(async () => {
  try {
    const health = await healthCheck()
    console.log('✓ Backend is healthy:', health.status)
    backendAvailable = true
  } catch (error) {
    console.warn('⊘ Backend is not running. Skipping integration tests.')
    console.warn('  Error:', (error as Error).message)
    backendAvailable = false
  }
})

describe('API Integration Tests', () => {
  describe('Health Check', () => {
    it('should return status ok', async () => {
      if (!backendAvailable) return
      const result = await healthCheck()
      expect(result.status).toBe('healthy')
    }, 5000)
  })

  describe('Paper Registration', () => {
    it('should register a paper from arXiv', async () => {
      if (!backendAvailable) return
      const result = await registerPaper('1706.03762')
      expect(result.status).toBe('success')
      expect(result.data.id).toBeTruthy()
      testPaperId = result.data.id
      console.log('✓ Registered paper:', testPaperId)
    }, 30000)

    it('should handle invalid arXiv ID', async () => {
      if (!backendAvailable) return
      await expect(registerPaper('invalid-id')).rejects.toThrow()
    }, 10000)
  })

  describe('Paper Search', () => {
    it('should search papers semantically', async () => {
      if (!backendAvailable) return
      const results = await searchPapers('transformer attention mechanism')
      expect(Array.isArray(results)).toBe(true)
      expect(results.length).toBeGreaterThan(0)
      console.log(`✓ Found ${results.length} results`)
    }, 15000)

    it('should search papers by keyword', async () => {
      if (!backendAvailable) return
      const results = await searchPapers('attention', 'keyword')
      expect(Array.isArray(results)).toBe(true)
    }, 15000)
  })

  describe('Paper Pagination', () => {
    it('should get paginated papers', async () => {
      if (!backendAvailable) return
      const page1 = await getPapersPaginated(0, 5)
      expect(Array.isArray(page1)).toBe(true)
      console.log(`✓ Got page 1 with ${page1.length} papers`)
    }, 10000)

    it('should get different pages', async () => {
      if (!backendAvailable) return
      const page1 = await getPapersPaginated(0, 2)
      const page2 = await getPapersPaginated(2, 2)
      expect(page1.length).toBeLessThanOrEqual(2)
      expect(page2.length).toBeLessThanOrEqual(2)
      console.log(`✓ Page 1: ${page1.length} papers, Page 2: ${page2.length} papers`)
    }, 10000)
  })

  describe('Tags', () => {
    it('should get all tags', async () => {
      if (!backendAvailable) return
      const tags = await getAllTags()
      expect(Array.isArray(tags)).toBe(true)
      console.log(`✓ Found ${tags.length} tags`)
    }, 10000)

    it('should create a new tag', async () => {
      if (!backendAvailable) return
      const testTagName = `Test Tag ${Date.now()}`
      const tag = await createTag(testTagName, '#ff0000')
      expect(tag).toHaveProperty('id')
      expect(tag.name).toBe(testTagName)
      testTagId = tag.id
      console.log(`✓ Created tag with ID ${testTagId}`)
    }, 10000)

    it('should handle duplicate tag name', async () => {
      if (!backendAvailable) return
      const duplicateName = `Duplicate Test ${Date.now()}`
      await createTag(duplicateName, '#00ff00')
      await expect(createTag(duplicateName, '#0000ff')).rejects.toThrow()
    }, 15000)
  })

  describe('Paper Deletion', () => {
    it('should delete a single paper', async () => {
      if (!backendAvailable || !testPaperId) return
      const result = await deletePaper(testPaperId)
      expect(result.status).toBe('success')
      console.log(`✓ Deleted paper ${testPaperId}`)
    }, 10000)

    it('should delete multiple papers', async () => {
      if (!backendAvailable) return
      // 既存の論文を再登録（テスト用）
      const paper1 = await registerPaper('1706.03762')  // Attention Is All You Need
      const paper2 = await registerPaper('1810.04805')  // BERT
      const idsToDelete = [paper1.data.id, paper2.data.id]
      const result = await deletePapers(idsToDelete)
      expect(result.status).toBe('success')
      expect(result.count).toBe(2)
      console.log(`✓ Deleted ${result.count} papers in batch`)
    }, 60000)
  })
})
