/// <reference types="vitest/config" />

import { fileURLToPath, URL } from 'node:url';
import tailwindcss from '@tailwindcss/vite';
import { tanstackRouter } from '@tanstack/router-plugin/vite';
import react from '@vitejs/plugin-react';
import { defineConfig, loadEnv } from 'vite';

function normalizeBasePath(value: string | undefined): string {
  const path = value?.trim() || '/';
  if (!path.startsWith('/') || path.includes('?') || path.includes('#')) {
    throw new Error('VITE_BASE_PATH must be an absolute URL path');
  }
  return path.endsWith('/') ? path : `${path}/`;
}

function resolveProxyTarget(value: string | undefined): string {
  const target = value?.trim() || 'http://127.0.0.1:8025';
  const url = new URL(target);
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new Error('SPA_API_PROXY_TARGET must use http or https');
  }
  return url.toString();
}

export default defineConfig(({ mode }) => {
  const values = loadEnv(mode, process.cwd(), '');
  const base = normalizeBasePath(values.VITE_BASE_PATH);
  const proxyTarget = resolveProxyTarget(values.SPA_API_PROXY_TARGET);

  return {
    base,
    plugins: [
      tanstackRouter({
        target: 'react',
        autoCodeSplitting: true,
      }),
      react(),
      tailwindcss(),
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      host: '127.0.0.1',
      port: 4173,
      strictPort: true,
      headers: {
        'Referrer-Policy': 'strict-origin-when-cross-origin',
        'X-Content-Type-Options': 'nosniff',
        'X-Frame-Options': 'DENY',
      },
      proxy: {
        '/api': {
          target: proxyTarget,
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, ''),
        },
      },
    },
    preview: {
      host: '127.0.0.1',
      port: 4174,
      strictPort: true,
    },
    build: {
      target: 'baseline-widely-available',
      outDir: 'dist',
      assetsDir: 'assets',
      cssCodeSplit: true,
      manifest: true,
      sourcemap: false,
      reportCompressedSize: true,
      chunkSizeWarningLimit: 500,
    },
    test: {
      environment: 'happy-dom',
      globals: true,
      setupFiles: ['./src/test/setup.ts'],
      css: true,
      coverage: {
        provider: 'v8',
        reporter: ['text', 'json-summary'],
        exclude: ['src/routeTree.gen.ts', 'src/main.tsx'],
      },
    },
  };
});
