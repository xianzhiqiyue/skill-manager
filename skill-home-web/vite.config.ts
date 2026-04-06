import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

function normalizeViteBasePath(value?: string | null) {
  const trimmed = value?.trim() || '';
  if (!trimmed || trimmed === '/') {
    return '/';
  }

  return `/${trimmed.replace(/^\/+|\/+$/g, '')}/`;
}

export default defineConfig(({ command }) => {
  const env = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env;
  const explicitBasePath = env?.VITE_APP_BASE_PATH;
  const defaultBasePath = command === 'build' ? '/skill-home/' : '/';

  return {
    base: normalizeViteBasePath(explicitBasePath || defaultBasePath),
    plugins: [react()],
    server: {
      host: '0.0.0.0',
      port: 4173,
    },
    test: {
      environment: 'jsdom',
      setupFiles: './src/test/setup.ts',
    },
  };
});
