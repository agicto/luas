import { defineConfig, devices } from '@playwright/test';

/**
 * End-to-end smoke tests against a running stack: the Next.js Web application, the Go API, and
 * PostgreSQL. Start the stack with `make dev` from the repository root, then run
 * `corepack pnpm e2e`. Set PLAYWRIGHT_CHANNEL=chrome to use an installed Chrome.
 */
export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.e2e.ts',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['list']] : 'list',
  timeout: 60_000,
  use: {
    baseURL: process.env.E2E_WEB_URL ?? 'http://localhost:3000',
    locale: 'en-US',
    trace: 'retain-on-failure',
    ...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}),
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
