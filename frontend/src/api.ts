// =============================================================================
// APIクライアント
// =============================================================================

import type {
  RegisterPaperResponse,
  SearchResult,
  SearchMode,
  Tag
} from './types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

// =============================================================================
// 論文登録API
// =============================================================================

export async function registerPaper(arxivId: string): Promise<RegisterPaperResponse> {
  const response = await fetch(`${API_BASE_URL}/api/papers`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ arxiv_id: arxivId }),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

// =============================================================================
// 論文検索API
// =============================================================================

export async function searchPapers(
  query: string,
  mode: SearchMode = 'semantic'
): Promise<SearchResult[]> {
  const params = new URLSearchParams({
    q: query,
    mode: mode,
  });

  const response = await fetch(`${API_BASE_URL}/api/search?${params}`);

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

// =============================================================================
// ヘルスチェックAPI
// =============================================================================

export async function healthCheck(): Promise<{ status: string }> {
  const response = await fetch(`${API_BASE_URL}/health`);
  return response.json();
}

// =============================================================================
// 全論文取得API（非推奨：getPapersPaginatedを使用してください）
// =============================================================================

export async function getAllPapers(): Promise<SearchResult[]> {
  const response = await fetch(`${API_BASE_URL}/api/papers`);

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

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

  const response = await fetch(`${API_BASE_URL}/api/papers?${params}`);

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

// =============================================================================
// 論文削除API
// =============================================================================

export async function deletePaper(id: string): Promise<{ status: string; message: string; id: string }> {
  const response = await fetch(`${API_BASE_URL}/api/papers/${id}`, {
    method: 'DELETE',
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

// =============================================================================
// 論文一括削除API
// =============================================================================

export async function deletePapers(ids: string[]): Promise<{ status: string; message: string; count: number }> {
  const response = await fetch(`${API_BASE_URL}/api/papers/batch-delete`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ ids }),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

// =============================================================================
// タグ関連API
// =============================================================================

export async function getAllTags(): Promise<Tag[]> {
  const response = await fetch(`${API_BASE_URL}/api/tags`);

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

export async function createTag(name: string, color: string): Promise<Tag> {
  const response = await fetch(`${API_BASE_URL}/api/tags`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ name, color }),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

export async function updateTag(id: number, name: string, color: string): Promise<{ status: string; message: string }> {
  const response = await fetch(`${API_BASE_URL}/api/tags/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ name, color }),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

export async function deleteTag(id: number): Promise<{ status: string; message: string }> {
  const response = await fetch(`${API_BASE_URL}/api/tags/${id}`, {
    method: 'DELETE',
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

export async function getPaperTags(paperId: string): Promise<Tag[]> {
  const response = await fetch(`${API_BASE_URL}/api/papers/${paperId}/tags`);

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}

export async function setPaperTags(paperId: string, tagIds: number[]): Promise<{ status: string; message: string }> {
  const response = await fetch(`${API_BASE_URL}/api/papers/${paperId}/tags`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ tag_ids: tagIds }),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`APIエラー: ${response.status} - ${errorText}`);
  }

  return response.json();
}
