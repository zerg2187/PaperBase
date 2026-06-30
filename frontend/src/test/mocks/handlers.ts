import { http, HttpResponse } from 'msw'

export const handlers = [
  // Health check
  http.get('http://localhost:8080/health', () => {
    return HttpResponse.json({ status: 'healthy' })
  }),

  // Auth status
  http.get('http://localhost:8080/api/auth/status', () => {
    return HttpResponse.json({
      role: 'admin',
      session_id: 'test-session-id',
    })
  }),

  // Pagination
  http.get('http://localhost:8080/api/papers', ({ request }) => {
    const url = new URL(request.url)
    const offset = parseInt(url.searchParams.get('offset') || '0')
    const limit = parseInt(url.searchParams.get('limit') || '10')

    const mockPapers = Array.from({ length: Math.min(limit, 3) }, (_, i) => ({
      id: `paper-${offset + i}`,
      title: `Test Paper ${offset + i + 1}`,
      authors: ['Author One', 'Author Two'],
      venue: 'Test Conference 2024',
      year: 2024,
      abstract: 'This is a test abstract.',
      bibtex: `@article{test${offset + i}, title={Test Paper ${offset + i + 1}}}`,
      similarity: 0.8 - i * 0.05,
      tags: [],
    }))

    return HttpResponse.json(mockPapers)
  }),

  // Search
  http.get('http://localhost:8080/api/search', () => {
    return HttpResponse.json([
      {
        id: 'search-result-1',
        title: 'Search Result Paper',
        authors: ['Search Author'],
        venue: 'Search Venue',
        year: 2024,
        abstract: 'Search abstract',
        bibtex: '@article{search}',
        similarity: 0.9,
        tags: [],
      },
    ])
  }),

  // Register
  http.post('http://localhost:8080/api/papers', async ({ request }) => {
    const body = (await request.json()) as { arxiv_id: string }
    return HttpResponse.json({
      status: 'success',
      message: 'Paper registered',
      data: { id: 'new-paper-' + body.arxiv_id, title: 'New Paper' },
    })
  }),

  // Tags
  http.get('http://localhost:8080/api/tags', () => {
    return HttpResponse.json([
      { id: 1, name: 'LLM', color: '#6366f1' },
      { id: 2, name: 'Vision', color: '#10b981' },
    ])
  }),

  // Delete papers
  http.post('http://localhost:8080/api/papers/batch-delete', () => {
    return HttpResponse.json({ status: 'success', message: 'Deleted', count: 1 })
  }),

  // Set paper tags
  http.put('http://localhost:8080/api/papers/:paperId/tags', () => {
    return HttpResponse.json({ status: 'success', message: 'Tags updated' })
  }),

  // Create tag
  http.post('http://localhost:8080/api/tags', async ({ request }) => {
    const body = (await request.json()) as { name: string; color: string }
    return HttpResponse.json({
      id: 3,
      name: body.name,
      color: body.color,
    })
  }),

  // Get paper tags
  http.get('http://localhost:8080/api/papers/:paperId/tags', () => {
    return HttpResponse.json([{ id: 1, name: 'LLM', color: '#6366f1' }])
  }),
]
