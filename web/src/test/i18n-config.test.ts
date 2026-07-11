import { afterEach, describe, expect, it, vi } from 'vitest';

type EnvOverrides = Record<string, string | undefined>;

const originalEnv = { ...process.env };
const managedEnvKeys = [
  'NEXT_PUBLIC_DEFAULT_LOCALE',
  'NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED',
  'SESSION_SECRET',
] as const;

async function loadI18nConfig(overrides: EnvOverrides = {}) {
  vi.resetModules();

  for (const key of managedEnvKeys) {
    delete process.env[key];
  }

  process.env.SESSION_SECRET = 'luas-test-session-secret-at-least-32-chars';

  for (const [key, value] of Object.entries(overrides)) {
    if (value === undefined) {
      delete process.env[key];
    } else {
      process.env[key] = value;
    }
  }

  return import('@/i18n/config');
}

afterEach(() => {
  vi.resetModules();

  for (const key of managedEnvKeys) {
    delete process.env[key];
  }

  for (const key of managedEnvKeys) {
    const value = originalEnv[key];

    if (value !== undefined) {
      process.env[key] = value;
    }
  }
});

describe('i18n config', () => {
  it('uses scaffold defaults when locale env vars are not set', async () => {
    const config = await loadI18nConfig({
      NEXT_PUBLIC_DEFAULT_LOCALE: undefined,
      NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED: undefined,
    });

    expect(config.defaultLocale).toBe('zh-Hans');
    expect(config.isLocaleSwitcherEnabled).toBe(true);
  });

  it('reads default locale and switcher visibility from typed env config', async () => {
    const config = await loadI18nConfig({
      NEXT_PUBLIC_DEFAULT_LOCALE: 'en-US',
      NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED: 'false',
    });

    expect(config.defaultLocale).toBe('en-US');
    expect(config.isLocaleSwitcherEnabled).toBe(false);
  });

  it('rejects unsupported default locales', async () => {
    await expect(
      loadI18nConfig({
        NEXT_PUBLIC_DEFAULT_LOCALE: 'fr-FR',
      })
    ).rejects.toThrow(
      'Invalid public environment variable: NEXT_PUBLIC_DEFAULT_LOCALE'
    );
  });
});
