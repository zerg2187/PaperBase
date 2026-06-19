// =============================================================================
// 型定義（バックエンドAPIと同期）
// =============================================================================

export interface Paper {
  id: string;
  title: string;
  authors: string[];
  venue: string;
  year: number;
  abstract: string;
  bibtex: string;
  similarity?: number;
  tags?: Tag[];
}

export interface Tag {
  id: number;
  name: string;
  color: string;
}

export interface RegisterPaperRequest {
  arxiv_id: string;
}

export interface RegisterPaperResponse {
  status: string;
  message: string;
  data: {
    id: string;
    title: string;
  };
}

export interface SearchResult {
  id: string;
  title: string;
  authors: string[];
  venue: string;
  year: number;
  abstract: string;
  bibtex: string;
  similarity?: number;
  tags?: Tag[];
}

export type SearchMode = 'semantic' | 'keyword';

export interface CreateTagRequest {
  name: string;
  color: string;
}

export interface UpdateTagRequest {
  name: string;
  color: string;
}

export interface SetPaperTagsRequest {
  tag_ids: number[];
}

export interface DeletePapersRequest {
  ids: string[];
}
