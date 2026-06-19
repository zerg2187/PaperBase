// =============================================================================
// MSW (Mock Service Worker) ハンドラー
// =============================================================================

import { http, HttpResponse } from 'msw';
import type { RegisterPaperResponse, SearchResult } from '../types';

// モックデータ
const mockPapers: SearchResult[] = [
  {
    id: '2406.11717',
    title: 'Refusal in Language Models Is Mediated by a Single Direction',
    authors: ['Andy Arditi', 'Oscar Obeso', 'Aaquib Syed', 'Daniel Paleka', 'Nina Panickssery', 'Wes Gurnee', 'Neel Nanda'],
    venue: 'NeurIPS',
    abstract: 'Conversational large language models are fine-tuned for both instruction-following and safety...',
    bibtex: '@inproceedings{arditi2024refusal,\n  title={Refusal in Language Models Is Mediated by a Single Direction},\n  author={Arditi, Andy and Obeso, Oscar and Syed, Aaquib and Paleka, Daniel and Panickssery, Nina and Gurnee, Wes and Nanda, Neel},\n  booktitle={Advances in Neural Information Processing Systems},\n  year={2024}\n}',
    similarity: 0.95,
  },
  {
    id: '2305.14345',
    title: 'Llama 2: Open Foundation and Fine-Tuned Chat Models',
    authors: ['Hugo Touvron', 'Louis Martin', 'Kevin Stone', 'Peter Albert', 'Alyssia Bakht'],
    venue: 'Preprint',
    abstract: 'We introduce Llama 2, a collection of pretrained and fine-tuned large language models...',
    bibtex: '@article{touvron2023llama,\n  title={Llama 2: Open Foundation and Fine-Tuned Chat Models},\n  author={Touvron, Hugo and Martin, Louis and Stone, Kevin and Albert, Peter and Bakht, Alyssia},\n  journal={arXiv preprint arXiv:2305.14345},\n  year={2023}\n}',
  },
];

export const handlers = [
  // ヘルスチェック
  http.get('/health', () => {
    return HttpResponse.json({ status: 'healthy' });
  }),

  // 論文登録API
  http.post('/api/papers', async ({ request }) => {
    const body = await request.json() as { arxiv_id: string };

    // バリデーション
    if (!body.arxiv_id || body.arxiv_id === '') {
      return HttpResponse.json(
        { error: 'arxiv_id は必須です' },
        { status: 400 }
      );
    }

    // モックレスポンス
    const response: RegisterPaperResponse = {
      status: 'success',
      message: 'Paper registered successfully',
      data: {
        id: body.arxiv_id,
        title: mockPapers[0].title,
      },
    };

    return HttpResponse.json(response, { status: 201 });
  }),

  // 論文検索API
  http.get('/api/search', ({ request }) => {
    const url = new URL(request.url);
    const query = url.searchParams.get('q');
    const mode = url.searchParams.get('mode') || 'semantic';

    // バリデーション
    if (!query) {
      return HttpResponse.json(
        { error: 'クエリパラメータ q は必須です' },
        { status: 400 }
      );
    }

    // キーワード検索のモック（簡易版）
    if (mode === 'keyword') {
      const filtered = mockPapers.filter(p =>
        p.title.toLowerCase().includes(query.toLowerCase()) ||
        p.abstract.toLowerCase().includes(query.toLowerCase())
      );
      return HttpResponse.json(filtered);
    }

    // セマンティック検索のモック（全件返す）
    return HttpResponse.json(mockPapers);
  }),
];
