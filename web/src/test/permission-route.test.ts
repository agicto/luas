import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const cookieStore = vi.hoisted(() => ({ get: vi.fn(), set: vi.fn() }));
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
  'API_UPSTREAM_MAX_RESPONSE_BYTES',
  'API_UPSTREAM_TIMEOUT_MS',
  'API_UPSTREAM_URL',
  'MOCK_BFF_ENABLED',
  'NEXT_PUBLIC_API_URL',
  'NEXT_PUBLIC_APP_URL',
  'NEXT_PUBLIC_OPTIONAL_FEATURES',
] as const;

describe('permission browser route boundary', () => {
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
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'organization,permission';
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

  it('requires both optional Web features', async () => {
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'organization';
    process.env.MOCK_BFF_ENABLED = 'true';
    const { getPermissionContextRoute } = await import(
      '@/features/permission/server/permission-route'
    );

    const response = await getPermissionContextRoute(request('/api/permission-context', {
      organizationId: '1',
    }));

    expect(response.status).toBe(503);
    await expect(response.json()).resolves.toMatchObject({
      error_code: 'COMMON.SERVICE_UNAVAILABLE',
    });
  });

  it('authenticates before exposing organization-context validation', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(null);
    const { getPermissionContextRoute } = await import(
      '@/features/permission/server/permission-route'
    );

    const response = await getPermissionContextRoute(request('/api/permission-context'));

    expect(response.status).toBe(401);
    await expect(response.json()).resolves.toMatchObject({ error_code: 'AUTH.UNAUTHORIZED' });
  });

  it('serves owner effective permissions with private tenant-varying headers', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(owner());
    const { getPermissionContextRoute } = await import(
      '@/features/permission/server/permission-route'
    );

    const response = await getPermissionContextRoute(request('/api/permission-context', {
      organizationId: '1',
    }));

    expect(response.status).toBe(200);
    expect(response.headers.get('cache-control')).toBe('private, no-store');
    expect(response.headers.get('vary')).toContain('Organization-Id');
    await expect(response.json()).resolves.toMatchObject({
      data: { organization_id: 1, is_owner: true },
    });
  });

  it('rejects cross-origin role creation before reading session or body', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const { createAccessRoleRoute } = await import(
      '@/features/permission/server/permission-route'
    );

    const response = await createAccessRoleRoute(request('/api/access-roles', {
      method: 'POST',
      origin: 'https://evil.example',
      organizationId: '1',
      body: JSON.stringify({ name: 'Must not exist', slug: 'must-not-exist', permissions: [] }),
    }));

    expect(response.status).toBe(403);
    expect(getSessionUser).not.toHaveBeenCalled();
  });

  it('forwards assignment replacement through the fixed PUT path and tenant header', async () => {
    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    cookieStore.get.mockReturnValue({ value: compactJwt() });
    fetchMock.mockResolvedValueOnce(Response.json({
      code: 0,
      message: 'success',
      data: { member_id: 3, access_role_ids: [1] },
    }));
    const { replaceMemberAccessRolesRoute } = await import(
      '@/features/permission/server/permission-route'
    );

    const response = await replaceMemberAccessRolesRoute(
      request('/api/organization-members/3/access-roles', {
        method: 'PUT',
        organizationId: '1',
        body: JSON.stringify({ access_role_ids: [1] }),
      }),
      '3'
    );

    expect(response.status).toBe(200);
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe('https://api.example.com/v1/organization-members/3/access-roles');
    expect(init?.method).toBe('PUT');
    expect(new Headers(init?.headers).get('organization-id')).toBe('1');
    expect(JSON.parse(String(init?.body))).toEqual({ access_role_ids: [1] });
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
  if (options.organizationId) result.headers.set('organization-id', options.organizationId);
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

function owner() {
  return { id: 'demo-admin', email: 'admin@example.com', name: 'Admin User' };
}
