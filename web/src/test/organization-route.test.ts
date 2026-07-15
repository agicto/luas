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
    getSessionUser.mockResolvedValue(mockOwner());
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
    getSessionUser.mockResolvedValue(mockOwner());
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
    getSessionUser.mockResolvedValue(mockOwner());
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

  it('serves a PII-minimized member directory from the mock lifecycle', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockOwner());
    const { listOrganizationMembersRoute } = await import(
      '@/features/organization/server/organization-lifecycle-route'
    );

    const response = await listOrganizationMembersRoute(
      request('/api/organizations/1/members'),
      '1'
    );
    const body = await response.json();

    expect(response.status).toBe(200);
    expect(response.headers.get('cache-control')).toBe('private, no-store');
    expect(body).toMatchObject({ meta: { total: 3 } });
    expect(JSON.stringify(body)).not.toContain('admin@example.com');
  });

  it('creates a token-free mock invitation after same-origin and auth guards', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockOwner());
    const { createOrganizationInvitationRoute } = await import(
      '@/features/organization/server/organization-lifecycle-route'
    );

    const response = await createOrganizationInvitationRoute(
      request('/api/organizations/1/invitations', {
        method: 'POST',
        body: JSON.stringify({ email: 'route.member@example.com', role: 'member' }),
      }),
      '1'
    );
    const body = await response.json();

    expect(response.status).toBe(201);
    expect(body).toMatchObject({
      data: {
        invitation: { email: 'route.member@example.com', status: 'pending' },
        email_send_status: 'not_configured',
      },
    });
    expect(JSON.stringify(body)).not.toContain('oinv_');
  });

  it('rejects cross-origin invitation acceptance before reading the bearer token', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const { acceptOrganizationInvitationRoute } = await import(
      '@/features/organization/server/organization-lifecycle-route'
    );

    const response = await acceptOrganizationInvitationRoute(
      request('/api/organization-invitations/accept', {
        method: 'POST',
        origin: 'https://evil.example',
        body: JSON.stringify({ token: 'must-not-be-read' }),
      })
    );

    expect(response.status).toBe(403);
    expect(getSessionUser).not.toHaveBeenCalled();
  });

  it('forwards invitation acceptance only to the fixed production path', async () => {
    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    cookieStore.get.mockReturnValue({ value: compactJwt() });
    fetchMock.mockResolvedValueOnce(
      Response.json({
        code: 0,
        message: 'success',
        data: organizationView(),
      })
    );
    const { acceptOrganizationInvitationRoute } = await import(
      '@/features/organization/server/organization-lifecycle-route'
    );

    const response = await acceptOrganizationInvitationRoute(
      request('/api/organization-invitations/accept', {
        method: 'POST',
        body: JSON.stringify({ token: 'oinv_private' }),
      })
    );

    expect(response.status).toBe(200);
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe(
      'https://api.example.com/v1/organization-invitations/accept'
    );
    expect(init?.method).toBe('POST');
    expect(JSON.parse(String(init?.body))).toEqual({ token: 'oinv_private' });
  });

  it('forwards ownership transfer through one fixed production path', async () => {
    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    cookieStore.get.mockReturnValue({ value: compactJwt() });
    fetchMock.mockResolvedValueOnce(
      Response.json({
        code: 0,
        message: 'success',
        data: {
          previous_owner: memberView(1, 'admin'),
          new_owner: memberView(3, 'owner'),
        },
      })
    );
    const { transferOrganizationOwnershipRoute } = await import(
      '@/features/organization/server/organization-lifecycle-route'
    );

    const response = await transferOrganizationOwnershipRoute(
      request('/api/organizations/1/ownership-transfer', {
        method: 'POST',
        body: JSON.stringify({ new_owner_member_id: 3 }),
      }),
      '1'
    );

    expect(response.status).toBe(200);
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe('https://api.example.com/v1/organizations/1/ownership-transfer');
    expect(init?.method).toBe('POST');
    expect(JSON.parse(String(init?.body))).toEqual({ new_owner_member_id: 3 });
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

function mockOwner() {
  return {
    id: 'demo-admin',
    email: 'admin@example.com',
    name: 'Admin User',
  };
}

function memberView(id: number, role: 'admin' | 'owner') {
  return {
    id,
    user_id: id,
    username: `member-${id}`,
    nickname: `Member ${id}`,
    role,
    joined_at: '2026-07-15T10:00:00Z',
    updated_at: '2026-07-15T10:00:00Z',
  };
}

function organizationView() {
  return {
    id: 1,
    name: 'Luas Demo',
    slug: 'luas-demo',
    role: 'member',
    created_at: '2026-07-15T10:00:00Z',
    updated_at: '2026-07-15T10:00:00Z',
  };
}
