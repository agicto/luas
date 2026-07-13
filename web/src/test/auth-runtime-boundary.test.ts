import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const sourceRoot = resolve(process.cwd(), 'src');

function source(path: string): string {
  return readFileSync(resolve(sourceRoot, path), 'utf8');
}

describe('protected auth runtime boundary', () => {
  it('resolves auth on the server before constructing protected providers', () => {
    const layout = source('app/(protected)/layout.tsx');

    expect(layout).toContain('resolveAuthBootstrap');
    expect(layout).toContain('bootstrap={bootstrap}');
  });

  it('keeps mock session helpers out of the client provider graph', () => {
    const provider = source('providers/auth-provider.tsx');
    const authenticatedProviders = source('providers/authenticated-providers.tsx');

    expect(provider).not.toContain("@/features/auth/server/");
    expect(provider).not.toContain('next/headers');
    expect(authenticatedProviders).not.toContain("@/features/auth/server/");
    expect(authenticatedProviders).not.toContain('next/headers');
  });

  it('makes middleware enforcement conditional on the auth runtime mode', () => {
    const middleware = readFileSync(resolve(process.cwd(), 'middleware.ts'), 'utf8');

    expect(middleware).toContain('getAuthRuntimeMode');
    expect(middleware).toContain("!== 'mock-session'");
  });
});
