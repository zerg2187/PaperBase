// =============================================================================
// APIクライアント
// =============================================================================

import type {
  RegisterPaperResponse,
  SearchResult,
  SearchMode,
  Tag,
  AuthState,
} from './types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

// Admin トークンはメモリ上に保持（セキュリティのため localStorage には保存しない）
let adminToken: string | null = null;

export function setAdminToken(token: string | null) {
  adminToken = token;
}

export function getAdminToken(): string | null {
  return adminToken;
}

function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (adminToken) {
    headers['Authorization'] = `Bearer ${adminToken}`;
  }
  return headers;
}

function getAuthHeaderOnly(): Record<string, string> {
  return adminToken ? { Authorization: `Bearer ${adminToken}` } : {};
}

async function fetchJSON<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: 'include',
    ...options,
    headers: {
      ...(options.headers || {}),
      ...getAuthHeaders(),
    },
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

async function fetchJSONWithAdminHeader<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: 'include',
    ...options,
    headers: {
      ...(options.headers || {}),
      ...getAuthHeaderOnly(),
    },
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

// =============================================================================
// 認証 API
// =============================================================================

export async function loginAdmin(token: string): Promise<{ status: string; role: string }> {
  const response = await fetch(`${API_BASE_URL}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ token }),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`認証エラー: ${response.status} - ${errorText}`);
  }

  adminToken = token;
  return response.json();
}

export async function logoutAdmin(): Promise<{ status: string; message: string }> {
  return fetchJSON('/api/auth/logout', { method: 'POST' });
}

export async function getMe(): Promise<AuthState> {
  return fetchJSONWithAdminHeader('/api/auth/me');
}

export async function getGuestStatus(): Promise<{ role: string; session_id: string; remaining_paper_count?: number }> {
  return fetchJSONWithAdminHeader('/api/auth/status');
}

// =============================================================================
// 論文登録API
// =============================================================================

export async function registerPaper(arxivId: string): Promise<RegisterPaperResponse> {
  return fetchJSON('/api/papers', {
    method: 'POST',
    body: JSON.stringify({ arxiv_id: arxivId }),
  });
}

// =============================================================================
// 論文検索API
// =============================================================================

export async function searchPapers(
  query: string,
  mode: SearchMode = 'semantic'
): Promise<SearchResult[]> {
  const params = new URLSearchParams({ q: query, mode });
  return fetchJSONWithAdminHeader(`/api/search?${params}`);
}

// =============================================================================
// ヘルスチェックAPI
// =============================================================================

export async function healthCheck(): Promise<{ status: string }> {
  const response = await fetch(`${API_BASE_URL}/health`, {
    credentials: 'include',
  });
  return response.json();
}

// =============================================================================
// 論文一覧取得API（ページネーション対応）
// =============================================================================

export async function getPapersPaginated(
  offset: number = 0,
  limit: number = 10,
  tagId: number | null = null
): Promise<SearchResult[]> {
  const params = new URLSearchParams({
    offset: offset.toString(),
    limit: limit.toString(),
  });

  if (tagId !== null) {
    params.set('tag_id', tagId.toString());
  }

  return fetchJSONWithAdminHeader(`/api/papers?${params}`);
}

// =============================================================================
// 論文削除API
// =============================================================================

export async function deletePaper(id: string): Promise<{ status: string; message: string; id: string }> {
  return fetchJSONWithAdminHeader(`/api/papers/${id}`, { method: 'DELETE' });
}

// =============================================================================
// 論文一括削除API
// =============================================================================

export async function deletePapers(ids: string[]): Promise<{ status: string; message: string; count: number }> {
  return fetchJSON('/api/papers/batch-delete', {
    method: 'POST',
    body: JSON.stringify({ ids }),
  });
}

// =============================================================================
// タグ関連API
// =============================================================================

export async function getAllTags(): Promise<Tag[]> {
  return fetchJSON('/api/tags');
}

export async function createTag(name: string, color: string): Promise<Tag> {
  return fetchJSON('/api/tags', {
    method: 'POST',
    body: JSON.stringify({ name, color }),
  });
}

export async function updateTag(id: number, name: string, color: string): Promise<{ status: string; message: string }> {
  return fetchJSON(`/api/tags/${id}`, {
    method: 'PUT',
    body: JSON.stringify({ name, color }),
  });
}

export async function deleteTag(id: number): Promise<{ status: string; message: string }> {
  return fetchJSON(`/api/tags/${id}`, { method: 'DELETE' });
}

export async function getPaperTags(paperId: string): Promise<Tag[]> {
  return fetchJSON(`/api/papers/${paperId}/tags`);
}

export async function setPaperTags(paperId: string, tagIds: number[]): Promise<{ status: string; message: string }> {
  return fetchJSON(`/api/papers/${paperId}/tags`, {
    method: 'PUT',
    body: JSON.stringify({ tag_ids: tagIds }),
  });
}
