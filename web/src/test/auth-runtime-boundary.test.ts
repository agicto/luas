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

  it('keeps mock credentials in a server-only module', () => {
    const authConfig = source('config/auth.ts');
    const loginForm = source('features/auth/components/login-form.tsx');
    const mockIdentity = source('features/auth/server/mock-identity.ts');

    expect(authConfig).not.toContain('admin@example.com');
    expect(authConfig).not.toContain('admin123');
    expect(authConfig).not.toContain('demoUser');
    expect(loginForm).not.toContain('admin@example.com');
    expect(loginForm).not.toContain('admin123');
    expect(mockIdentity).toContain("import 'server-only'");
    expect(mockIdentity).toContain('admin@example.com');
    expect(mockIdentity).toContain('admin123');
  });

  it('uses the shared mock cookie policy in middleware and exact-scope logout', () => {
    const middleware = readFileSync(resolve(process.cwd(), 'middleware.ts'), 'utf8');
    const session = source('features/auth/server/session.ts');

    expect(middleware).toContain('getMockSessionCookieName()');
    expect(session).toContain('createMockSessionCookie(signed)');
    expect(session).toContain('cookieStore.set(createExpiredMockSessionCookie())');
    expect(session).not.toContain('cookieStore.delete(');
  });
});
