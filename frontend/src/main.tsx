import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'

// MSW (開発環境のみ) - 無効化中
// async function prepare() {
//   if (import.meta.env.DEV) {
//     const { worker } = await import('./mocks/browser');
//     await worker.start({
//       onUnhandledRequest: 'bypass',
//     });
//     console.log('🔶 MSW initialized');
//   }
// }

// prepare().then(() => {
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
// });
