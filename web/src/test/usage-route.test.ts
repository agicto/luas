import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const cookieStore = vi.hoisted(() => ({ get: vi.fn(), set: vi.fn() }));
const getSessionUser = vi.hoisted(() => vi.fn());

vi.mock('next/headers', () => ({
  cookies: async () => cookieStore,
  headers: async () => new Headers(),
}));
vi.mock('@/features/auth/server/session', async importOriginal => {
  const original = await importOriginal<typeof import('@/features/auth/server/session')>();
  return { ...original, getSessionUser };
});

const originalEnv = { ...process.env };
const managedKeys = [
  'API_ADAPTER_ENABLED',
  'API_UPSTREAM_MAX_RESPONSE_BYTES',
  'API_UPSTREAM_TIMEOUT_MS',
  'API_UPSTREAM_URL',
  'MOCK_BFF_ENABLED',
  'NEXT_PUBLIC_API_URL',
  'NEXT_PUBLIC_APP_URL',
  'NEXT_PUBLIC_OPTIONAL_FEATURES',
] as const;

describe('usage browser route boundary', () => {
  let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;

  beforeEach(() => {
    vi.resetModules();
    fetchMock = vi.fn<typeof fetch>();
    vi.stubGlobal('fetch', fetchMock);
    cookieStore.get.mockReset();
    cookieStore.set.mockReset();
    getSessionUser.mockReset();
    process.env.NEXT_PUBLIC_API_URL = '/api';
    process.env.NEXT_PUBLIC_APP_URL = 'https://app.example.com';
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'organization,usage';
    process.env.API_UPSTREAM_TIMEOUT_MS = '5000';
    process.env.API_UPSTREAM_MAX_RESPONSE_BYTES = '1048576';
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.resetModules();
    for (const key of managedKeys) {
      delete process.env[key];
      if (originalEnv[key] !== undefined) process.env[key] = originalEnv[key];
    }
  });

  it('keeps the optional route unavailable when usage is disabled', async () => {
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'organization';
    process.env.MOCK_BFF_ENABLED = 'true';
    const route = await import('@/features/usage/server/usage-route');

    const response = await route.userUsageRoute(request('/api/usage/user'));

    expect(response.status).toBe(503);
    expect(response.headers.get('cache-control')).toBe('no-store');
    await expect(response.json()).resolves.toMatchObject({
      error_code: 'COMMON.SERVICE_UNAVAILABLE',
    });
  });

  it('serves finite private user usage from the mock BFF', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('demo-owner'));
    const route = await import('@/features/usage/server/usage-route');

    const response = await route.userUsageRoute(request('/api/usage/user'));

    expect(response.status).toBe(200);
    expect(response.headers.get('cache-control')).toBe('private, no-store');
    expect(response.headers.get('pragma')).toBe('no-cache');
    expect(response.headers.get('vary')).toContain('Cookie');
    const body = await response.json();
    expect(body.data).toHaveLength(5);
    expect(body.data[0]).toMatchObject({
      scope: 'user',
      metric: 'api.requests',
      quota_source: 'override',
    });
    expect(JSON.stringify(body)).not.toContain('event_id');
  });

  it('allows organization managers and rejects ordinary members', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const route = await import('@/features/usage/server/usage-route');

    getSessionUser.mockResolvedValue(mockUser('demo-member'));
    const forbidden = await route.organizationUsageRoute(
      request('/api/organization-usage', { headers: { 'Organization-Id': '1' } })
    );
    expect(forbidden.status).toBe(403);
    expect(forbidden.headers.get('vary')).toContain('Organization-Id');
    await expect(forbidden.json()).resolves.toMatchObject({
      error_code: 'PERMISSION.DENIED',
    });

    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const allowed = await route.organizationUsageRoute(
      request('/api/organization-usage', { headers: { 'Organization-Id': '1' } })
    );
    expect(allowed.status).toBe(200);
    const allowedBody = await allowed.json();
    expect(allowedBody.data).toHaveLength(5);
    expect(allowedBody.data[0]).toMatchObject({
      scope: 'organization',
      metric: 'api.requests',
    });
  });

  it('forwards only fixed production paths and explicit organization context', async () => {
    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    cookieStore.get.mockReturnValue({ value: compactJwt() });
    fetchMock
      .mockResolvedValueOnce(Response.json({ code: 0, message: 'success', data: [] }))
      .mockResolvedValueOnce(Response.json({ code: 0, message: 'success', data: [] }));
    const route = await import('@/features/usage/server/usage-route');

    await route.userUsageRoute(request('/api/usage/user'));
    await route.organizationUsageRoute(
      request('/api/organization-usage', { headers: { 'Organization-Id': '7' } })
    );

    expect(String(fetchMock.mock.calls[0][0])).toBe('https://api.example.com/v1/usage/user');
    expect(String(fetchMock.mock.calls[1][0])).toBe(
      'https://api.example.com/v1/organization-usage'
    );
    expect(new Headers(fetchMock.mock.calls[1][1]?.headers).get('organization-id')).toBe('7');
  });
});

function request(path: string, options: { headers?: Record<string, string> } = {}): Request {
  return new Request(`https://app.example.com${path}`, { headers: options.headers });
}

function mockUser(id: string) {
  return { id, email: `${id}@example.com`, name: id };
}

function compactJwt(): string {
  return 'eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature';
}
