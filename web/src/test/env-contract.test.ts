import { readdirSync, readFileSync, statSync } from 'node:fs';
import { relative, resolve } from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';

const sourceRoot = resolve(process.cwd(), 'src');
const allowedProcessEnvFiles = new Set([
  'config/env.ts',
  'test/setup.ts',
]);
const originalEnv = { ...process.env };
const managedEnvKeys = [
  'NEXT_PHASE',
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

  it('requires SESSION_SECRET in production runtime', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    delete process.env.NEXT_PHASE;
    delete process.env.SESSION_SECRET;

    await expect(import('@/config/env')).rejects.toThrow(
      'SESSION_SECRET must be set in production runtime'
    );
  });

  it('allows production builds to run without runtime-only SESSION_SECRET', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    process.env.NEXT_PHASE = 'phase-production-build';
    delete process.env.SESSION_SECRET;

    const config = await import('@/config/env');

    expect(config.env.NODE_ENV).toBe('production');
    expect(config.env.SESSION_SECRET).toBeUndefined();
  });

  it('rejects weak SESSION_SECRET values before runtime starts', async () => {
    vi.resetModules();
    vi.stubEnv('NODE_ENV', 'production');
    delete process.env.NEXT_PHASE;
    process.env.SESSION_SECRET = 'too-short';

    await expect(import('@/config/env')).rejects.toThrow('Invalid environment variables');
  });
});
