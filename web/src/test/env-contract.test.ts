import { readdirSync, readFileSync, statSync } from 'node:fs';
import { relative, resolve } from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';

const sourceRoot = resolve(process.cwd(), 'src');
const allowedProcessEnvFiles = new Set([
  'config/env.ts',
  'config/server-env.ts',
  'test/setup.ts',
]);
const originalEnv = { ...process.env };
const managedEnvKeys = [
  'AUTH_ADAPTER_ENABLED',
  'AUTH_API_TIMEOUT_MS',
  'AUTH_API_URL',
  'AUTH_CLIENT_IP_HEADER',
  'MOCK_BFF_ENABLED',
  'NEXT_PHASE',
  'NEXT_PUBLIC_API_URL',
  'NEXT_PUBLIC_APP_URL',
  'SESSION_SECRET',
] as const;

function listSourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((entry) => {
    const path = resolve(dir, entry);
    const stat = statSync(path);

    if (stat.isDirectory()) {
      return listSourceFiles(path);
    }

    return path.endsWith('.ts') || path.endsWith('.tsx') ? [path] : [];
  });
}

function isAllowedProcessEnvFile(path: string): boolean {
  const relativePath = relative(sourceRoot, path);

  return (
    allowedProcessEnvFiles.has(relativePath) ||
    relativePath.endsWith('.test.ts') ||
    relativePath.endsWith('.test.tsx') ||
    relativePath.includes('__tests__/')
  );
}

describe('environment config contract', () => {
  afterEach(() => {
    vi.resetModules();
    vi.unstubAllEnvs();

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

  it('keeps production source code behind src/config/env.ts', () => {
    const offenders = listSourceFiles(sourceRoot)
      .filter((path) => !isAllowedProcessEnvFile(path))
      .filter((path) => readFileSync(path, 'utf8').includes('process.env'))
      .map((path) => relative(sourceRoot, path));

    expect(offenders).toEqual([]);
  });

  it('keeps the client-safe env usable without server secrets', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    delete process.env.NEXT_PHASE;
    delete process.env.AUTH_ADAPTER_ENABLED;
    delete process.env.AUTH_API_TIMEOUT_MS;
    delete process.env.AUTH_API_URL;
    delete process.env.AUTH_CLIENT_IP_HEADER;
    delete process.env.MOCK_BFF_ENABLED;
    delete process.env.SESSION_SECRET;

    const config = await import('@/config/env');

    expect(config.env.NODE_ENV).toBe('production');
    expect(config.env).not.toHaveProperty('SESSION_SECRET');
    expect(config.env).not.toHaveProperty('MOCK_BFF_ENABLED');
    expect(config.env).not.toHaveProperty('AUTH_API_URL');
    expect(config.env).not.toHaveProperty('AUTH_CLIENT_IP_HEADER');
  });

  it('requires a private upstream when the production adapter is enabled', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    vi.stubEnv('NEXT_PUBLIC_API_URL', '/api');
    vi.stubEnv('NEXT_PUBLIC_APP_URL', 'https://app.example.com');
    delete process.env.NEXT_PHASE;
    process.env.AUTH_ADAPTER_ENABLED = 'true';
    delete process.env.AUTH_API_URL;

    await expect(import('@/config/server-env')).rejects.toThrow(
      'AUTH_API_URL must be set when AUTH_ADAPTER_ENABLED=true'
    );
  });

  it('requires an explicit ingress-owned client IP header in production', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    vi.stubEnv('NEXT_PUBLIC_API_URL', '/api');
    vi.stubEnv('NEXT_PUBLIC_APP_URL', 'https://app.example.com');
    delete process.env.NEXT_PHASE;
    process.env.AUTH_ADAPTER_ENABLED = 'true';
    process.env.AUTH_API_URL = 'http://api:8025/v1';
    delete process.env.AUTH_CLIENT_IP_HEADER;

    await expect(import('@/config/server-env')).rejects.toThrow(
      'AUTH_CLIENT_IP_HEADER must be set when AUTH_ADAPTER_ENABLED=true in production runtime'
    );
  });

  it('accepts a complete production auth adapter configuration', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    vi.stubEnv('NEXT_PUBLIC_API_URL', '/api');
    vi.stubEnv('NEXT_PUBLIC_APP_URL', 'https://app.example.com');
    delete process.env.NEXT_PHASE;
    process.env.AUTH_ADAPTER_ENABLED = 'true';
    process.env.AUTH_API_URL = 'http://api:8025/v1';
    process.env.AUTH_API_TIMEOUT_MS = '7500';
    process.env.AUTH_CLIENT_IP_HEADER = 'X-Forwarded-For';

    const config = await import('@/config/server-env');

    expect(config.serverEnv).toMatchObject({
      AUTH_ADAPTER_ENABLED: true,
      AUTH_API_URL: 'http://api:8025/v1',
      AUTH_API_TIMEOUT_MS: 7500,
      AUTH_CLIENT_IP_HEADER: 'x-forwarded-for',
    });
  });

  it('rejects adapter mode when browser auth bypasses the same-origin route', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    vi.stubEnv('NEXT_PUBLIC_API_URL', 'https://api.example.com/v1');
    vi.stubEnv('NEXT_PUBLIC_APP_URL', 'https://app.example.com');
    delete process.env.NEXT_PHASE;
    process.env.AUTH_ADAPTER_ENABLED = 'true';
    process.env.AUTH_API_URL = 'http://api:8025/v1';
    process.env.AUTH_CLIENT_IP_HEADER = 'x-forwarded-for';

    await expect(import('@/config/server-env')).rejects.toThrow(
      'NEXT_PUBLIC_API_URL must target the same-origin /api route when AUTH_ADAPTER_ENABLED=true'
    );
  });

  it('requires SESSION_SECRET when production explicitly enables the mock BFF', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    delete process.env.NEXT_PHASE;
    process.env.MOCK_BFF_ENABLED = 'true';
    delete process.env.SESSION_SECRET;

    await expect(import('@/config/server-env')).rejects.toThrow(
      'SESSION_SECRET must be set when MOCK_BFF_ENABLED=true in production runtime'
    );
  });

  it('allows production runtime without mock credentials when the mock BFF is disabled', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    delete process.env.NEXT_PHASE;
    process.env.MOCK_BFF_ENABLED = 'false';
    process.env.AUTH_ADAPTER_ENABLED = 'false';
    delete process.env.SESSION_SECRET;

    const config = await import('@/config/server-env');

    expect(config.serverEnv.MOCK_BFF_ENABLED).toBe(false);
    expect(config.serverEnv.SESSION_SECRET).toBeUndefined();
  });

  it('allows production builds to omit runtime-only SESSION_SECRET', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    process.env.NEXT_PHASE = 'phase-production-build';
    process.env.MOCK_BFF_ENABLED = 'true';
    process.env.AUTH_ADAPTER_ENABLED = 'true';
    delete process.env.AUTH_API_URL;
    delete process.env.AUTH_CLIENT_IP_HEADER;
    delete process.env.SESSION_SECRET;

    const config = await import('@/config/server-env');

    expect(config.serverEnv.MOCK_BFF_ENABLED).toBe(true);
    expect(config.serverEnv.AUTH_ADAPTER_ENABLED).toBe(true);
    expect(config.serverEnv.AUTH_API_URL).toBeUndefined();
    expect(config.serverEnv.SESSION_SECRET).toBeUndefined();
  });

  it('rejects weak SESSION_SECRET values before runtime starts', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    delete process.env.NEXT_PHASE;
    process.env.MOCK_BFF_ENABLED = 'true';
    process.env.SESSION_SECRET = 'too-short';

    await expect(import('@/config/server-env')).rejects.toThrow(
      'Invalid server environment variables'
    );
  });

  it('keeps server-only values out of the client env module and config barrel', () => {
    const clientEnv = readFileSync(resolve(sourceRoot, 'config/env.ts'), 'utf8');
    const configBarrel = readFileSync(resolve(sourceRoot, 'config/index.ts'), 'utf8');
    const serverEnv = readFileSync(resolve(sourceRoot, 'config/server-env.ts'), 'utf8');
    const rootLayout = readFileSync(resolve(sourceRoot, 'app/layout.tsx'), 'utf8');

    expect(clientEnv).not.toContain('SESSION_SECRET');
    expect(clientEnv).not.toContain('MOCK_BFF_ENABLED');
    expect(clientEnv).not.toContain('AUTH_ADAPTER_ENABLED');
    expect(clientEnv).not.toContain('AUTH_API_URL');
    expect(clientEnv).not.toContain('AUTH_CLIENT_IP_HEADER');
    expect(clientEnv).not.toMatch(/from ['"]zod['"]/);
    expect(configBarrel).not.toContain('server-env');
    expect(serverEnv).toContain("import 'server-only'");
    expect(rootLayout).toContain("@/config/server-env");
  });
});
