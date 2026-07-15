import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const cookieStore = vi.hoisted(() => ({
  get: vi.fn(),
  set: vi.fn(),
}));
const getSessionUser = vi.hoisted(() => vi.fn());

vi.mock('next/headers', () => ({
  cookies: async () => cookieStore,
  headers: async () => new Headers(),
}));
vi.mock('@/features/auth/server/session', async (importOriginal) => {
  const original = await importOriginal<typeof import('@/features/auth/server/session')>();
  return { ...original, getSessionUser };
});

const originalEnv = { ...process.env };
const managedKeys = [
  'API_ADAPTER_ENABLED',
  'API_CLIENT_IP_HEADER',
  'API_UPSTREAM_MAX_RESPONSE_BYTES',
  'API_UPSTREAM_TIMEOUT_MS',
  'API_UPSTREAM_URL',
  'MOCK_BFF_ENABLED',
  'NEXT_PUBLIC_API_URL',
  'NEXT_PUBLIC_APP_URL',
  'NEXT_PUBLIC_OPTIONAL_FEATURES',
] as const;

describe('organization browser route boundary', () => {
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
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'organization';
    process.env.API_UPSTREAM_TIMEOUT_MS = '5000';
    process.env.API_UPSTREAM_MAX_RESPONSE_BYTES = '1048576';
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
    vi.resetModules();
    for (const key of managedKeys) {
      delete process.env[key];
      if (originalEnv[key] !== undefined) process.env[key] = originalEnv[key];
    }
  });

  it('keeps the optional route unavailable when its Web feature is disabled', async () => {
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = '';
    process.env.MOCK_BFF_ENABLED = 'true';
    const { listOrganizationsRoute } = await import(
      '@/features/organization/server/organization-route'
    );

    const response = await listOrganizationsRoute(request('/api/organizations'));

    expect(response.status).toBe(503);
    await expect(response.json()).resolves.toMatchObject({
      error_code: 'COMMON.SERVICE_UNAVAILABLE',
    });
  });

  it('serves a contract-shaped page from the development mock BFF', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue({ id: 'demo-admin' });
    const { listOrganizationsRoute } = await import(
      '@/features/organization/server/organization-route'
    );

    const response = await listOrganizationsRoute(
      request('/api/organizations?page=1&per_page=15')
    );

    expect(response.status).toBe(200);
    expect(response.headers.get('cache-control')).toBe('private, no-store');
    expect(response.headers.get('vary')).toContain('Cookie');
    await expect(response.json()).resolves.toMatchObject({
      code: 0,
      data: [{ id: 1, role: 'owner' }],
      meta: { current_page: 1, total: 1 },
    });
  });

  it('rejects a cross-origin create before authentication or state access', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const { createOrganizationRoute } = await import(
      '@/features/organization/server/organization-route'
    );

    const crossOriginRequest = request('/api/organizations', {
      method: 'POST',
      origin: 'https://evil.example',
      body: JSON.stringify({ name: 'Should not exist' }),
    });
    expect(crossOriginRequest.headers.get('sec-fetch-site')).toBe('cross-site');
    const response = await createOrganizationRoute(crossOriginRequest);

    expect(response.status).toBe(403);
    expect(getSessionUser).not.toHaveBeenCalled();
  });

  it('authenticates before exposing payload or organization-context validation', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(null);
    const { createOrganizationRoute, resolveOrganizationContextRoute } = await import(
      '@/features/organization/server/organization-route'
    );

    const createResponse = await createOrganizationRoute(
      request('/api/organizations', { method: 'POST', body: '{' })
    );
    const contextResponse = await resolveOrganizationContextRoute(
      request('/api/organization-context')
    );

    expect(createResponse.status).toBe(401);
    expect(contextResponse.status).toBe(401);
    await expect(createResponse.json()).resolves.toMatchObject({
      error_code: 'AUTH.UNAUTHORIZED',
    });
    await expect(contextResponse.json()).resolves.toMatchObject({
      error_code: 'AUTH.UNAUTHORIZED',
    });
  });

  it('returns the canonical public error for an oversized organization body', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue({ id: 'demo-admin' });
    const { createOrganizationRoute } = await import(
      '@/features/organization/server/organization-route'
    );

    const response = await createOrganizationRoute(
      request('/api/organizations', {
        method: 'POST',
        body: JSON.stringify({ name: 'a'.repeat(70_000) }),
      })
    );

    expect(response.status).toBe(413);
    await expect(response.json()).resolves.toMatchObject({
      code: 413,
      error_code: 'COMMON.REQUEST_TOO_LARGE',
    });
  });

  it('distinguishes a missing, malformed, and valid organization context', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue({ id: 'demo-admin' });
    const { resolveOrganizationContextRoute } = await import(
      '@/features/organization/server/organization-route'
    );

    const missing = await resolveOrganizationContextRoute(request('/api/organization-context'));
    const malformed = await resolveOrganizationContextRoute(
      request('/api/organization-context', { organizationId: '01' })
    );
    const valid = await resolveOrganizationContextRoute(
      request('/api/organization-context', { organizationId: '1' })
    );

    await expect(missing.json()).resolves.toMatchObject({
      error_code: 'ORGANIZATION.CONTEXT_REQUIRED',
    });
    await expect(malformed.json()).resolves.toMatchObject({
      error_code: 'ORGANIZATION.CONTEXT_INVALID',
    });
    expect(valid.headers.get('vary')).toContain('Organization-Id');
    await expect(valid.json()).resolves.toMatchObject({
      data: { organization_id: 1, role: 'owner' },
    });
  });

  it('forwards production list calls through the fixed authenticated adapter', async () => {
    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    cookieStore.get.mockReturnValue({ value: compactJwt() });
    fetchMock.mockResolvedValueOnce(
      Response.json(
        {
          code: 0,
          message: 'success',
          data: [],
          meta: {
            current_page: 1,
            per_page: 15,
            total: 0,
            last_page: 1,
            from: 0,
            to: 0,
          },
          links: { first: '', last: '', prev: null, next: null },
        },
        { headers: { vary: 'Organization-Id' } }
      )
    );
    const { listOrganizationsRoute } = await import(
      '@/features/organization/server/organization-route'
    );

    const response = await listOrganizationsRoute(request('/api/organizations'));

    expect(response.status).toBe(200);
    expect(response.headers.get('cache-control')).toBe('private, no-store');
    expect(response.headers.get('vary')).toContain('Organization-Id');
    expect(response.headers.get('vary')).toContain('Cookie');
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe('https://api.example.com/v1/organizations?page=1&per_page=15');
    expect(new Headers(init?.headers).get('authorization')).toBe(`Bearer ${compactJwt()}`);
    await expect(response.json()).resolves.toMatchObject({ meta: { total: 0 } });
  });
});

function request(
  path: string,
  options: {
    method?: string;
    origin?: string;
    body?: BodyInit;
    organizationId?: string;
  } = {}
): Request {
  const result = new Request(`https://app.example.com${path}`, {
    method: options.method ?? 'GET',
    body: options.body,
  });
  if (options.origin) {
    result.headers.set('origin', options.origin);
    result.headers.set('sec-fetch-site', 'cross-site');
  }
  if (options.organizationId) {
    result.headers.set('organization-id', options.organizationId);
  }
  if (options.body) result.headers.set('content-type', 'application/json');
  return result;
}

function compactJwt(): string {
  return [
    Buffer.from('{}').toString('base64url'),
    Buffer.from('{"exp":4102444800}').toString('base64url'),
    'signature',
  ].join('.');
}
