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

describe('asset browser route boundary', () => {
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
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'asset';
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

  it('runs the isolated mock upload, completion, download, and deletion lifecycle', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('alice'));
    const route = await import('@/features/asset/server/asset-route');
    const bytes = new TextEncoder().encode('%PDF-1.7\nprivate\n');

    const createdResponse = await route.createAssetUploadIntentRoute(
      jsonRequest('/api/assets/upload-intents', {
        idempotency_key: 'asset-route-1',
        original_name: 'private.pdf',
        media_type: 'application/pdf',
        size_bytes: bytes.byteLength,
      })
    );
    expect(createdResponse.status).toBe(201);
    const created = await createdResponse.json();
    const assetId = created.data.asset.id as string;
    const uploadToken = tokenFromUrl(created.data.upload.url as string);

    const uploadRequest = new Request(
      `https://app.example.com/api/asset-transfers/${uploadToken}`,
      {
        method: 'PUT',
        headers: {
          'content-length': String(bytes.byteLength),
          'content-type': 'application/pdf',
          origin: 'https://app.example.com',
          'sec-fetch-site': 'same-origin',
        },
        body: bytes,
      }
    );
    expect((await route.acceptAssetTransferRoute(uploadRequest, uploadToken)).status).toBe(204);

    const completeResponse = await route.completeAssetRoute(
      mutationRequest(`/api/assets/${assetId}/complete`),
      assetId
    );
    await expect(completeResponse.json()).resolves.toMatchObject({
      data: { id: assetId, status: 'ready' },
    });

    const listed = await (await route.listAssetsRoute(request('/api/assets?status=ready'))).json();
    expect(listed.data).toHaveLength(1);
    expect(JSON.stringify(listed)).not.toContain('checksum');
    expect(JSON.stringify(listed)).not.toContain('object_key');

    const grantResponse = await route.createAssetDownloadGrantRoute(
      mutationRequest(`/api/assets/${assetId}/download-grant`),
      assetId
    );
    const grant = await grantResponse.json();
    const downloadResponse = route.downloadAssetTransferRoute(tokenFromUrl(grant.data.url));
    expect(downloadResponse.headers.get('content-disposition')).toContain('attachment');
    expect(new Uint8Array(await downloadResponse.arrayBuffer())).toEqual(bytes);

    getSessionUser.mockResolvedValue(mockUser('bob'));
    expect(
      (
        await route.createAssetDownloadGrantRoute(
          mutationRequest(`/api/assets/${assetId}/download-grant`),
          assetId
        )
      ).status
    ).toBe(404);

    getSessionUser.mockResolvedValue(mockUser('alice'));
    expect(
      (await route.deleteAssetRoute(mutationRequest(`/api/assets/${assetId}`, 'DELETE'), assetId))
        .status
    ).toBe(204);
    expect(
      (await route.deleteAssetRoute(mutationRequest(`/api/assets/${assetId}`, 'DELETE'), assetId))
        .status
    ).toBe(204);
  });

  it('rejects cross-origin intent creation before auth or body parsing', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const route = await import('@/features/asset/server/asset-route');
    const request = new Request('https://app.example.com/api/assets/upload-intents', {
      method: 'POST',
      body: '{',
    });
    request.headers.set('origin', 'https://evil.example');
    request.headers.set('sec-fetch-site', 'cross-site');
    const response = await route.createAssetUploadIntentRoute(request);
    expect(response.status).toBe(403);
    expect(getSessionUser).not.toHaveBeenCalled();
  });

  it('forwards only fixed production asset paths', async () => {
    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    cookieStore.get.mockReturnValue({ value: opaqueCredential() });
    fetchMock
      .mockResolvedValueOnce(Response.json(pageEnvelope([])))
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    const route = await import('@/features/asset/server/asset-route');
    const id = '019bf6d8-17c5-7a98-a084-6d45793f5f0c';

    await route.listAssetsRoute(request('/api/assets?page=2&per_page=50&status=ready&ignored=x'));
    await route.deleteAssetRoute(mutationRequest(`/api/assets/${id}`, 'DELETE'), id);

    expect(String(fetchMock.mock.calls[0][0])).toBe(
      'https://api.example.com/v1/assets?page=2&per_page=50&status=ready'
    );
    expect(String(fetchMock.mock.calls[1][0])).toBe(`https://api.example.com/v1/assets/${id}`);
  });
});

function request(path: string): Request {
  return new Request(`https://app.example.com${path}`);
}

function mutationRequest(path: string, method = 'POST'): Request {
  return new Request(`https://app.example.com${path}`, {
    method,
    headers: { origin: 'https://app.example.com', 'sec-fetch-site': 'same-origin' },
  });
}

function jsonRequest(path: string, body: unknown): Request {
  return new Request(`https://app.example.com${path}`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      origin: 'https://app.example.com',
      'sec-fetch-site': 'same-origin',
    },
    body: JSON.stringify(body),
  });
}

function tokenFromUrl(value: string): string {
  return new URL(value).pathname.split('/').at(-1) ?? '';
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
      per_page: 25,
      total: data.length,
      last_page: 1,
      from: data.length ? 1 : 0,
      to: data.length,
    },
    links: { first: '', last: '', prev: null, next: null },
  };
}
