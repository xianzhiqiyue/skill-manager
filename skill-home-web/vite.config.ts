import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

function normalizeViteBasePath(value?: string | null) {
  const trimmed = value?.trim() || '';
  if (!trimmed || trimmed === '/') {
    return '/';
  }

  return `/${trimmed.replace(/^\/+|\/+$/g, '')}/`;
}

export default defineConfig({
  base: normalizeViteBasePath(
    (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env?.VITE_APP_BASE_PATH,
  ),
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 4173,
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
});
