// =============================================================================
// MSW Node.js設定（テスト用）
// =============================================================================

import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);
