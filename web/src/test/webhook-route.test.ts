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

describe('webhook browser route boundary', () => {
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
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'organization,webhook';
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

  it('keeps the optional route unavailable unless organization and webhook are selected', async () => {
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'organization';
    process.env.MOCK_BFF_ENABLED = 'true';
    const route = await import('@/features/webhook/server/webhook-route');

    const response = await route.webhookEventTypesRoute(
      request('/api/webhook-event-types', { organizationId: '1' })
    );

    expect(response.status).toBe(503);
    expect(response.headers.get('cache-control')).toBe('private, no-store');
    await expect(response.json()).resolves.toMatchObject({
      error_code: 'COMMON.SERVICE_UNAVAILABLE',
    });
  });

  it('requires an organization manager without revealing cross-organization resources', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const route = await import('@/features/webhook/server/webhook-route');

    getSessionUser.mockResolvedValue(mockUser('demo-member'));
    const forbidden = await route.listWebhookEndpointsRoute(
      request('/api/webhook-endpoints', { organizationId: '1' })
    );
    expect(forbidden.status).toBe(403);
    expect(forbidden.headers.get('cache-control')).toBe('private, no-store');
    expect(forbidden.headers.get('vary')).toContain('Organization-Id');
    await expect(forbidden.json()).resolves.toMatchObject({
      error_code: 'PERMISSION.DENIED',
    });

    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const hidden = await route.listWebhookEndpointsRoute(
      request('/api/webhook-endpoints', { organizationId: '999' })
    );
    expect(hidden.status).toBe(404);
    await expect(hidden.json()).resolves.toMatchObject({
      error_code: 'ORGANIZATION.NOT_FOUND',
    });
  });

  it('rejects cross-origin writes before authentication or body parsing', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const route = await import('@/features/webhook/server/webhook-route');

    const crossOriginRequest = request('/api/webhook-endpoints', {
      method: 'POST',
      body: '{',
      organizationId: '1',
      origin: 'https://evil.example',
    });
    expect(crossOriginRequest.headers.get('sec-fetch-site')).toBe('cross-site');
    const response = await route.createWebhookEndpointRoute(crossOriginRequest);

    expect(response.status).toBe(403);
    expect(response.headers.get('cache-control')).toBe('private, no-store');
    expect(getSessionUser).not.toHaveBeenCalled();
  });

  it('preserves one-time secrets, CAS versions, and secret-free endpoint lists', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const route = await import('@/features/webhook/server/webhook-route');

    const created = await route.createWebhookEndpointRoute(
      request('/api/webhook-endpoints', {
        method: 'POST',
        organizationId: '1',
        body: endpointBody('Consumer'),
      })
    );
    expect(created.status).toBe(201);
    expect(created.headers.get('etag')).toBe('"webhook-endpoint-v1"');
    const createdBody = await created.json();
    expect(createdBody.data.signing_secret).toMatch(/^whsec_[A-Za-z0-9+/]+={0,2}$/u);
    expect(createdBody.data.endpoint).toMatchObject({
      id: 1,
      organization_id: 1,
      version: 1,
      status: 'active',
      disabled_reason: '',
    });

    const listed = await route.listWebhookEndpointsRoute(
      request('/api/webhook-endpoints', { organizationId: '1' })
    );
    const listedText = await listed.text();
    expect(listedText).not.toContain('signing_secret');
    expect(listedText).not.toContain('ciphertext');
    expect(JSON.parse(listedText)).toMatchObject({
      data: [{ id: 1, name: 'Consumer', version: 1 }],
      meta: { total: 1 },
    });

    const missingPrecondition = await route.updateWebhookEndpointRoute(
      request('/api/webhook-endpoints/1', {
        method: 'PATCH',
        organizationId: '1',
        body: endpointBody('Updated'),
      }),
      '1'
    );
    expect(missingPrecondition.status).toBe(428);
    await expect(missingPrecondition.json()).resolves.toMatchObject({
      error_code: 'WEBHOOK.PRECONDITION_REQUIRED',
    });

    const updated = await route.updateWebhookEndpointRoute(
      request('/api/webhook-endpoints/1', {
        method: 'PATCH',
        organizationId: '1',
        ifMatch: '"webhook-endpoint-v1"',
        body: endpointBody('Updated'),
      }),
      '1'
    );
    expect(updated.headers.get('etag')).toBe('"webhook-endpoint-v2"');

    const stale = await route.updateWebhookEndpointRoute(
      request('/api/webhook-endpoints/1', {
        method: 'PATCH',
        organizationId: '1',
        ifMatch: '"webhook-endpoint-v1"',
        body: endpointBody('Stale'),
      }),
      '1'
    );
    expect(stale.status).toBe(409);
    await expect(stale.json()).resolves.toMatchObject({
      error_code: 'WEBHOOK.ENDPOINT_VERSION_CONFLICT',
    });

    const rotated = await route.rotateWebhookEndpointSecretRoute(
      request('/api/webhook-endpoints/1/secret-rotations', {
        method: 'POST',
        organizationId: '1',
        ifMatch: '"webhook-endpoint-v2"',
      }),
      '1'
    );
    expect(rotated.status).toBe(201);
    const rotatedBody = await rotated.json();
    expect(rotatedBody.data.signing_secret).not.toBe(createdBody.data.signing_secret);
    expect(rotatedBody.data.endpoint).toMatchObject({ version: 3, secret_version: 2 });
  });

  it('keeps mock test delivery idempotent, terminal, minimized, and network-free', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const route = await import('@/features/webhook/server/webhook-route');
    await route.createWebhookEndpointRoute(
      request('/api/webhook-endpoints', {
        method: 'POST',
        organizationId: '1',
        body: endpointBody('Consumer'),
      })
    );

    const testRequest = () =>
      request('/api/webhook-endpoints/1/tests', {
        method: 'POST',
        organizationId: '1',
        idempotencyKey: 'stable-test-001',
      });
    const first = await route.testWebhookEndpointRoute(testRequest(), '1');
    const second = await route.testWebhookEndpointRoute(testRequest(), '1');
    const firstBody = await first.json();
    const secondBody = await second.json();

    expect(first.status).toBe(202);
    expect(second.status).toBe(202);
    expect(secondBody.data.id).toBe(firstBody.data.id);
    expect(firstBody.data).toMatchObject({
      status: 'canceled',
      attempt_count: 0,
      failure_code: 'WEBHOOK.MOCK_NOT_DELIVERED',
      http_status: null,
    });
    expect(JSON.stringify(firstBody)).not.toContain('https://hooks.example.com');
    expect(fetchMock).not.toHaveBeenCalled();

    const attempts = await route.listWebhookAttemptsRoute(
      request(`/api/webhook-deliveries/${firstBody.data.id}/attempts`, {
        organizationId: '1',
      }),
      String(firstBody.data.id)
    );
    await expect(attempts.json()).resolves.toMatchObject({ data: [], meta: { total: 0 } });
  });

  it('forwards only fixed paths and reviewed conditional and idempotency headers', async () => {
    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    cookieStore.get.mockReturnValue({ value: compactJwt() });
    fetchMock.mockResolvedValue(Response.json({ code: 0, message: 'success', data: { id: 1 } }));
    const route = await import('@/features/webhook/server/webhook-route');

    await route.createWebhookEndpointRoute(
      request('/api/webhook-endpoints', {
        method: 'POST',
        organizationId: '7',
        body: endpointBody('Consumer'),
      })
    );
    await route.updateWebhookEndpointRoute(
      request('/api/webhook-endpoints/11', {
        method: 'PATCH',
        organizationId: '7',
        ifMatch: '"webhook-endpoint-v4"',
        body: endpointBody('Updated'),
      }),
      '11'
    );
    await route.testWebhookEndpointRoute(
      request('/api/webhook-endpoints/11/tests', {
        method: 'POST',
        organizationId: '7',
        idempotencyKey: 'adapter-test-001',
      }),
      '11'
    );
    await route.listWebhookDeliveriesRoute(
      request('/api/webhook-deliveries?page=2&per_page=25&endpoint_id=11&status=failed', {
        organizationId: '7',
      })
    );

    expect(fetchMock.mock.calls.map(call => String(call[0]))).toEqual([
      'https://api.example.com/v1/webhook-endpoints',
      'https://api.example.com/v1/webhook-endpoints/11',
      'https://api.example.com/v1/webhook-endpoints/11/tests',
      'https://api.example.com/v1/webhook-deliveries?page=2&per_page=25&endpoint_id=11&status=failed',
    ]);
    const createHeaders = new Headers(fetchMock.mock.calls[0][1]?.headers);
    const updateHeaders = new Headers(fetchMock.mock.calls[1][1]?.headers);
    const testHeaders = new Headers(fetchMock.mock.calls[2][1]?.headers);
    expect(createHeaders.get('organization-id')).toBe('7');
    expect(updateHeaders.get('if-match')).toBe('"webhook-endpoint-v4"');
    expect(testHeaders.get('idempotency-key')).toBe('adapter-test-001');
    expect(testHeaders.get('cookie')).toBeNull();
  });
});

function request(
  path: string,
  options: {
    body?: string;
    idempotencyKey?: string;
    ifMatch?: string;
    method?: string;
    organizationId?: string;
    origin?: string;
  } = {}
): Request {
  const headers = new Headers();
  if (options.body !== undefined) headers.set('content-type', 'application/json');
  if (options.organizationId) headers.set('organization-id', options.organizationId);
  if (options.ifMatch) headers.set('if-match', options.ifMatch);
  if (options.idempotencyKey) headers.set('idempotency-key', options.idempotencyKey);
  const result = new Request(`https://app.example.com${path}`, {
    method: options.method ?? 'GET',
    body: options.body,
    headers,
  });
  if (options.origin) {
    result.headers.set('origin', options.origin);
    result.headers.set('sec-fetch-site', 'cross-site');
  }
  return result;
}

function endpointBody(name: string): string {
  return JSON.stringify({
    name,
    url: 'https://hooks.example.com/luas',
    event_types: ['webhook.test'],
  });
}

function mockUser(id: string) {
  return { id, email: `${id}@example.com`, name: id };
}

function compactJwt(): string {
  return 'eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.signature';
}
