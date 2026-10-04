import { defineConfig, devices } from '@playwright/test';

/**
 * End-to-end smoke tests against a running stack: the Admin Console, the Go API, and PostgreSQL.
 * Start the stack with `make dev` from the repository root, then run `corepack pnpm e2e`.
 * Set PLAYWRIGHT_CHANNEL=chrome to use an installed Chrome instead of a downloaded browser.
 */
export default defineConfig({
  testDir: './e2e',
  testMatch: '**/*.e2e.ts',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['list']] : 'list',
  timeout: 30_000,
  use: {
    baseURL: process.env.E2E_ADMIN_URL ?? 'http://127.0.0.1:4173',
    locale: 'en-US',
    trace: 'retain-on-failure',
    ...(process.env.PLAYWRIGHT_CHANNEL ? { channel: process.env.PLAYWRIGHT_CHANNEL } : {}),
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
