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
] as const;

describe('API key browser route boundary', () => {
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

  it('creates, lists, and idempotently revokes a token-free mock key', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser());
    const { createApiKeyRoute, listApiKeysRoute, revokeApiKeyRoute } =
      await import('@/features/api-key/server/api-key-route');

    const createdResponse = await createApiKeyRoute(
      request('/api/api-keys', {
        method: 'POST',
        body: JSON.stringify({ name: 'Deploy', scopes: [' Models:Read '] }),
      })
    );
    const created = await createdResponse.json();
    const apiKeyId = created.data.api_key.id as number;
    expect(createdResponse.status).toBe(201);
    expect(created.data.plaintext_key).toMatch(/^luas_/);

    const listedResponse = await listApiKeysRoute(request('/api/api-keys'));
    const listed = await listedResponse.json();
    expect(listed.data[0]).toMatchObject({ id: apiKeyId, scopes: ['models:read'] });
    expect(JSON.stringify(listed)).not.toContain(created.data.plaintext_key);
    expect(JSON.stringify(listed)).not.toContain('key_hash');

    expect(
      (
        await revokeApiKeyRoute(
          request(`/api/api-keys/${apiKeyId}`, { method: 'DELETE' }),
          String(apiKeyId)
        )
      ).status
    ).toBe(204);
    expect(
      (
        await revokeApiKeyRoute(
          request(`/api/api-keys/${apiKeyId}`, { method: 'DELETE' }),
          String(apiKeyId)
        )
      ).status
    ).toBe(204);
  });

  it('rejects cross-origin writes before authentication or body parsing', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const crossOriginRequest = request('/api/api-keys', {
      method: 'POST',
      origin: 'https://evil.example',
      body: '{',
    });
    expect(crossOriginRequest.headers.get('origin')).toBe('https://evil.example');
    expect(crossOriginRequest.headers.get('sec-fetch-site')).toBe('cross-site');
    const { guardSameOriginMutation } = await import('@/app/api/_shared/mock-bff');
    expect(guardSameOriginMutation(crossOriginRequest)?.status).toBe(403);
    const { createApiKeyRoute } = await import('@/features/api-key/server/api-key-route');
    const response = await createApiKeyRoute(crossOriginRequest);

    expect(response.status).toBe(403);
    expect(getSessionUser).not.toHaveBeenCalled();
  });

  it('authenticates before exposing malformed payload details', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(null);
    const { createApiKeyRoute } = await import('@/features/api-key/server/api-key-route');

    const response = await createApiKeyRoute(
      request('/api/api-keys', { method: 'POST', body: '{' })
    );

    expect(response.status).toBe(401);
    await expect(response.json()).resolves.toMatchObject({
      error_code: 'AUTH.UNAUTHORIZED',
    });
  });

  it('forwards only fixed production API key paths', async () => {
    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    cookieStore.get.mockReturnValue({ value: compactJwt() });
    fetchMock
      .mockResolvedValueOnce(Response.json(pageEnvelope([])))
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    const { listApiKeysRoute, revokeApiKeyRoute } =
      await import('@/features/api-key/server/api-key-route');

    await listApiKeysRoute(request('/api/api-keys?page=2&per_page=25'));
    await revokeApiKeyRoute(request('/api/api-keys/42', { method: 'DELETE' }), '42');

    expect(String(fetchMock.mock.calls[0][0])).toBe(
      'https://api.example.com/v1/api-keys?page=2&per_page=25'
    );
    expect(String(fetchMock.mock.calls[1][0])).toBe('https://api.example.com/v1/api-keys/42');
    expect(new Headers(fetchMock.mock.calls[0][1]?.headers).get('authorization')).toBe(
      `Bearer ${compactJwt()}`
    );
  });

  it('returns the canonical error for an oversized create body', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser());
    const { createApiKeyRoute } = await import('@/features/api-key/server/api-key-route');

    const response = await createApiKeyRoute(
      request('/api/api-keys', {
        method: 'POST',
        body: JSON.stringify({ name: 'x'.repeat(9_000), scopes: [] }),
      })
    );

    expect(response.status).toBe(413);
    await expect(response.json()).resolves.toMatchObject({
      error_code: 'COMMON.REQUEST_TOO_LARGE',
    });
  });
});

function request(
  path: string,
  options: { method?: string; body?: string; origin?: string } = {}
): Request {
  const result = new Request(`https://app.example.com${path}`, {
    method: options.method ?? 'GET',
    body: options.body,
  });
  if (options.body) result.headers.set('content-type', 'application/json');
  if (options.origin) {
    result.headers.set('origin', options.origin);
    result.headers.set('sec-fetch-site', 'cross-site');
  }
  return result;
}

function mockUser() {
  return { id: 'demo-admin', email: 'admin@example.com', name: 'Admin User' };
}

function compactJwt(): string {
  return 'eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature';
}

function pageEnvelope(data: unknown[]) {
  return {
    code: 0,
    message: 'success',
    data,
    meta: {
      current_page: 1,
      per_page: 15,
      total: data.length,
      last_page: 1,
      from: data.length ? 1 : 0,
      to: data.length,
    },
    links: { first: '', last: '', prev: null, next: null },
  };
}
