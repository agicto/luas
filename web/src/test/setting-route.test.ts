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

describe('setting browser route boundary', () => {
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
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'organization,setting';
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

  it('serves public settings with a stable ETag and bodyless revalidation', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const route = await import('@/features/setting/server/setting-route');

    const first = await route.publicSettingsRoute(request('/api/settings/public'));
    expect(first.status).toBe(200);
    expect(first.headers.get('cache-control')).toBe(
      'public, max-age=60, stale-while-revalidate=300'
    );
    const etag = first.headers.get('etag');
    expect(etag).toMatch(/^"settings-[a-f0-9]{64}"$/u);
    await expect(first.json()).resolves.toMatchObject({
      data: [{ key: 'branding.display_name' }, { key: 'localization.locale' }],
    });

    const second = await route.publicSettingsRoute(
      request('/api/settings/public', { headers: { 'If-None-Match': etag! } })
    );
    expect(second.status).toBe(304);
    expect(await second.text()).toBe('');
    expect(second.headers.get('etag')).toBe(etag);
  });

  it('preserves user isolation, monotonic versions, stale rejection, and reset', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('alice'));
    const route = await import('@/features/setting/server/setting-route');

    const created = await route.setUserSettingRoute(
      request('/api/settings/user/localization.locale', {
        method: 'PATCH',
        body: JSON.stringify({ value: 'zh-Hans' }),
        headers: { 'If-Match': '"setting-v0"' },
      }),
      'localization.locale'
    );
    expect(created.status).toBe(200);
    expect(created.headers.get('etag')).toBe('"setting-v1"');
    await expect(created.json()).resolves.toMatchObject({
      data: { value: 'zh-Hans', version: 1, source: 'override' },
    });

    const stale = await route.setUserSettingRoute(
      request('/api/settings/user/localization.locale', {
        method: 'PATCH',
        body: JSON.stringify({ value: 'en-US' }),
        headers: { 'If-Match': '"setting-v0"' },
      }),
      'localization.locale'
    );
    expect(stale.status).toBe(412);
    await expect(stale.json()).resolves.toMatchObject({
      error_code: 'SETTING.VERSION_CONFLICT',
    });

    getSessionUser.mockResolvedValue(mockUser('bob'));
    const bob = await route.userSettingsRoute(request('/api/settings/user'));
    const bobBody = await bob.json();
    expect(bobBody.data).toHaveLength(2);
    expect(bobBody.data[0]).toMatchObject({
      key: 'localization.locale',
      value: 'en-US',
      version: 0,
    });

    getSessionUser.mockResolvedValue(mockUser('alice'));
    const reset = await route.resetUserSettingRoute(
      request('/api/settings/user/localization.locale', {
        method: 'DELETE',
        headers: { 'If-Match': '"setting-v1"' },
      }),
      'localization.locale'
    );
    expect(reset.status).toBe(204);
    expect(reset.headers.get('etag')).toBe('"setting-v2"');

    const afterReset = await route.userSettingsRoute(request('/api/settings/user'));
    const afterResetBody = await afterReset.json();
    expect(afterResetBody.data).toHaveLength(2);
    expect(afterResetBody.data[0]).toMatchObject({
      key: 'localization.locale',
      value: 'en-US',
      version: 2,
      source: 'default',
    });
  });

  it('requires a canonical precondition and rejects invalid values before mutation', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('alice'));
    const route = await import('@/features/setting/server/setting-route');

    const missing = await route.setUserSettingRoute(
      request('/api/settings/user/localization.locale', {
        method: 'PATCH',
        body: JSON.stringify({ value: 'zh-Hans' }),
      }),
      'localization.locale'
    );
    expect(missing.status).toBe(428);
    await expect(missing.json()).resolves.toMatchObject({
      error_code: 'SETTING.PRECONDITION_REQUIRED',
    });

    const invalid = await route.setUserSettingRoute(
      request('/api/settings/user/localization.locale', {
        method: 'PATCH',
        body: JSON.stringify({ value: 'fr-FR' }),
        headers: { 'If-Match': '"setting-v0"' },
      }),
      'localization.locale'
    );
    expect(invalid.status).toBe(422);
    await expect(invalid.json()).resolves.toMatchObject({
      error_code: 'SETTING.INVALID_VALUE',
    });

    const malformed = await route.setUserSettingRoute(
      request('/api/settings/user/localization.locale', {
        method: 'PATCH',
        body: '{',
        headers: { 'If-Match': '"setting-v0"' },
      }),
      'localization.locale'
    );
    expect(malformed.status).toBe(400);
    expect(malformed.headers.get('cache-control')).toBe('private, no-store');
    expect(malformed.headers.get('vary')).toContain('Cookie');
  });

  it('lets organization members read but only owner/admin mutate', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const route = await import('@/features/setting/server/setting-route');

    getSessionUser.mockResolvedValue(mockUser('demo-member'));
    const read = await route.organizationSettingsRoute(
      request('/api/organization-settings', {
        headers: { 'Organization-Id': '1' },
      })
    );
    expect(read.status).toBe(200);
    expect(read.headers.get('vary')).toContain('Organization-Id');

    const forbidden = await route.setOrganizationSettingRoute(
      request('/api/organization-settings/localization.locale', {
        method: 'PATCH',
        body: JSON.stringify({ value: 'zh-Hans' }),
        headers: {
          'Organization-Id': '1',
          'If-Match': '"setting-v0"',
        },
      }),
      'localization.locale'
    );
    expect(forbidden.status).toBe(403);

    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const updated = await route.setOrganizationSettingRoute(
      request('/api/organization-settings/localization.locale', {
        method: 'PATCH',
        body: JSON.stringify({ value: 'zh-Hans' }),
        headers: {
          'Organization-Id': '1',
          'If-Match': '"setting-v0"',
        },
      }),
      'localization.locale'
    );
    expect(updated.status).toBe(200);
    await expect(updated.json()).resolves.toMatchObject({
      data: { scope: 'organization', value: 'zh-Hans', version: 1 },
    });
  });

  it('rejects cross-origin writes before authentication or body parsing', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const route = await import('@/features/setting/server/setting-route');
    const response = await route.setUserSettingRoute(
      request('/api/settings/user/localization.locale', {
        method: 'PATCH',
        body: '{',
        origin: 'https://evil.example',
        headers: { 'If-Match': '"setting-v0"' },
      }),
      'localization.locale'
    );

    expect(response.status).toBe(403);
    expect(response.headers.get('cache-control')).toBe('private, no-store');
    expect(response.headers.get('vary')).toContain('Cookie');
    expect(getSessionUser).not.toHaveBeenCalled();
  });

  it('forwards only fixed production paths and explicit conditional headers', async () => {
    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    cookieStore.get.mockReturnValue({ value: compactJwt() });
    fetchMock
      .mockResolvedValueOnce(
        new Response(null, {
          status: 304,
          headers: {
            etag: '"settings-upstream"',
            'cache-control': 'public, max-age=60',
          },
        })
      )
      .mockResolvedValueOnce(
        Response.json({
          code: 0,
          message: 'success',
          data: userSettings(),
        })
      )
      .mockResolvedValueOnce(
        Response.json({
          code: 0,
          message: 'success',
          data: userSettings()[0],
        })
      );
    const route = await import('@/features/setting/server/setting-route');

    await route.publicSettingsRoute(
      request('/api/settings/public', {
        headers: { 'If-None-Match': '"settings-upstream"' },
      })
    );
    await route.userSettingsRoute(request('/api/settings/user'));
    await route.setUserSettingRoute(
      request('/api/settings/user/localization.locale', {
        method: 'PATCH',
        body: JSON.stringify({ value: 'zh-Hans' }),
        headers: { 'If-Match': '"setting-v0"' },
      }),
      'localization.locale'
    );

    expect(String(fetchMock.mock.calls[0][0])).toBe('https://api.example.com/v1/settings/public');
    expect(String(fetchMock.mock.calls[1][0])).toBe('https://api.example.com/v1/settings/user');
    expect(String(fetchMock.mock.calls[2][0])).toBe(
      'https://api.example.com/v1/settings/user/localization.locale'
    );
    expect(new Headers(fetchMock.mock.calls[0][1]?.headers).get('if-none-match')).toBe(
      '"settings-upstream"'
    );
    expect(new Headers(fetchMock.mock.calls[2][1]?.headers).get('if-match')).toBe('"setting-v0"');
  });
});

function request(
  path: string,
  options: {
    method?: string;
    body?: string;
    origin?: string;
    headers?: Record<string, string>;
  } = {}
): Request {
  const result = new Request(`https://app.example.com${path}`, {
    method: options.method ?? 'GET',
    body: options.body,
    headers: options.headers,
  });
  if (options.body) result.headers.set('content-type', 'application/json');
  if (options.origin) {
    result.headers.set('origin', options.origin);
    result.headers.set('sec-fetch-site', 'cross-site');
  }
  return result;
}

function mockUser(id: string) {
  return { id, email: `${id}@example.com`, name: id };
}

function compactJwt(): string {
  return 'eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature';
}

function userSettings() {
  return [
    {
      scope: 'user',
      key: 'localization.locale',
      kind: 'enum',
      visibility: 'private',
      value: 'en-US',
      version: 0,
      source: 'default',
      options: ['en-US', 'zh-Hans'],
      updated_at: null,
    },
    {
      scope: 'user',
      key: 'localization.timezone',
      kind: 'timezone',
      visibility: 'private',
      value: 'UTC',
      version: 0,
      source: 'default',
      updated_at: null,
    },
  ];
}
