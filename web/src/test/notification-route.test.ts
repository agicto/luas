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

describe('notification browser route boundary', () => {
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
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'notification';
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

  it('lists, reads, marks through, and replaces preferences in mock mode', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const route = await import('@/features/notification/server/notification-route');

    const listedResponse = await route.listNotificationsRoute(request('/api/notifications'));
    const listed = await listedResponse.json();
    const newestId = listed.data[0].id as number;
    expect(listedResponse.status).toBe(200);
    expect(listed.data).toHaveLength(2);
    expect(listed.data[0].is_read).toBe(false);

    const updatedResponse = await route.replaceNotificationReadStateRoute(
      request(`/api/notifications/${newestId}`, {
        method: 'PATCH',
        body: JSON.stringify({ is_read: true }),
      }),
      String(newestId)
    );
    await expect(updatedResponse.json()).resolves.toMatchObject({
      data: { id: newestId, is_read: true },
    });

    const markedResponse = await route.markNotificationsReadRoute(
      request('/api/notification-read-state', {
        method: 'PUT',
        body: JSON.stringify({ through_id: newestId }),
      })
    );
    await expect(markedResponse.json()).resolves.toMatchObject({
      data: { unread_count: 0 },
    });

    const preferenceResponse = await route.replaceNotificationPreferenceRoute(
      request('/api/notification-preferences', {
        method: 'PUT',
        body: JSON.stringify({ in_app_enabled: true, email_enabled: false }),
      })
    );
    await expect(preferenceResponse.json()).resolves.toMatchObject({
      data: { in_app_enabled: true, email_enabled: false },
    });
  });

  it('does not reveal another mock user notification', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('alice'));
    const route = await import('@/features/notification/server/notification-route');
    const listed = await (await route.listNotificationsRoute(request('/api/notifications'))).json();
    const notificationId = listed.data[0].id as number;

    getSessionUser.mockResolvedValue(mockUser('bob'));
    const response = await route.replaceNotificationReadStateRoute(
      request(`/api/notifications/${notificationId}`, {
        method: 'PATCH',
        body: JSON.stringify({ is_read: true }),
      }),
      String(notificationId)
    );

    expect(response.status).toBe(404);
    await expect(response.json()).resolves.toMatchObject({
      error_code: 'NOTIFICATION.NOT_FOUND',
    });
  });

  it('rejects cross-origin writes before authentication or body parsing', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const route = await import('@/features/notification/server/notification-route');
    const response = await route.replaceNotificationPreferenceRoute(
      request('/api/notification-preferences', {
        method: 'PUT',
        origin: 'https://evil.example',
        body: '{',
      })
    );

    expect(response.status).toBe(403);
    expect(getSessionUser).not.toHaveBeenCalled();
  });

  it('rejects an invalid list filter instead of silently widening it', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const { listNotificationsRoute } = await import(
      '@/features/notification/server/notification-route'
    );
    const response = await listNotificationsRoute(
      request('/api/notifications?status=everything')
    );

    expect(response.status).toBe(422);
    await expect(response.json()).resolves.toMatchObject({
      error_code: 'COMMON.VALIDATION_FAILED',
    });
  });

  it('checks feature availability before authentication', async () => {
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = '';
    process.env.MOCK_BFF_ENABLED = 'true';
    const { listNotificationsRoute } = await import(
      '@/features/notification/server/notification-route'
    );
    const response = await listNotificationsRoute(request('/api/notifications'));

    expect(response.status).toBe(503);
    expect(getSessionUser).not.toHaveBeenCalled();
  });

  it('forwards only fixed production notification paths and canonical query values', async () => {
    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    cookieStore.get.mockReturnValue({ value: opaqueCredential() });
    fetchMock
      .mockResolvedValueOnce(Response.json(pageEnvelope([])))
      .mockResolvedValueOnce(Response.json({
        code: 0,
        message: 'success',
        data: { in_app_enabled: true, email_enabled: true },
      }));
    const route = await import('@/features/notification/server/notification-route');

    await route.listNotificationsRoute(
      request('/api/notifications?page=2&per_page=25&status=unread&ignored=value')
    );
    await route.getNotificationPreferenceRoute(request('/api/notification-preferences'));

    expect(String(fetchMock.mock.calls[0][0])).toBe(
      'https://api.example.com/v1/notifications?page=2&per_page=25&status=unread'
    );
    expect(String(fetchMock.mock.calls[1][0])).toBe(
      'https://api.example.com/v1/notification-preferences'
    );
    expect(new Headers(fetchMock.mock.calls[0][1]?.headers).get('authorization')).toBe(
      `Bearer ${opaqueCredential()}`
    );
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

function mockUser(id: string) {
  return { id, email: `${id}@example.com`, name: id };
}

function opaqueCredential(): string {
  return 'A'.repeat(43);
}

function pageEnvelope(data: unknown[]) {
  return {
    code: 0,
    message: 'success',
    data,
    meta: {
      current_page: 1,
      per_page: 10,
      total: data.length,
      last_page: 1,
      from: data.length ? 1 : 0,
      to: data.length,
    },
    links: { first: '', last: '', prev: null, next: null },
  };
}
