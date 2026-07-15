import { beforeEach, describe, expect, it, vi } from 'vitest';

import { GoApiClient } from '@/server/api-adapter/go-api-client';
import { ApiErrorCode } from '@/http/codes';

const randomId = '01234567-89ab-cdef-0123-456789abcdef';

describe('GoApiClient', () => {
  let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;
  let client: GoApiClient;

  beforeEach(() => {
    fetchMock = vi.fn<typeof fetch>();
    client = new GoApiClient(
      {
        apiUrl: 'https://api.example.com/v1',
        maxResponseBytes: 1_024,
        timeoutMs: 5_000,
        clientIpHeader: 'x-real-client-ip',
      },
      {
        fetch: fetchMock,
        randomUUID: () => randomId,
      }
    );
  });

  it('forwards only server-owned credentials and explicit organization context', async () => {
    fetchMock.mockResolvedValueOnce(successResponse({ organization_id: 42 }));
    const token = 'header.payload.signature';

    const result = await client.request({
      method: 'GET',
      path: 'organization-context',
      accessToken: token,
      organizationId: '42',
      incomingHeaders: new Headers({
        authorization: 'Bearer browser-controlled',
        cookie: 'browser=controlled',
        'organization-id': '99',
        'x-real-client-ip': '203.0.113.9',
        'x-request-id': 'req-org-1',
      }),
    });

    expect(result).toMatchObject({
      ok: true,
      data: {
        status: 200,
        envelope: {
          code: 0,
          data: { organization_id: 42 },
        },
      },
    });
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe('https://api.example.com/v1/organization-context');
    const headers = new Headers(init?.headers);
    expect(headers.get('authorization')).toBe(`Bearer ${token}`);
    expect(headers.get('organization-id')).toBe('42');
    expect(headers.get('cookie')).toBeNull();
    expect(headers.get('x-forwarded-for')).toBe('203.0.113.9');
    expect(headers.get('x-request-id')).toBe('req-org-1');
  });

  it('preserves pagination metadata and safe response headers', async () => {
    fetchMock.mockResolvedValueOnce(
      Response.json(
        {
          code: 0,
          message: 'success',
          data: [{ id: 42 }],
          meta: {
            current_page: 1,
            per_page: 15,
            total: 1,
            last_page: 1,
            from: 1,
            to: 1,
          },
          links: { first: '/v1/organizations?page=1', last: '', prev: null, next: null },
        },
        {
          headers: {
            vary: 'Organization-Id',
            'x-ratelimit-remaining': '99',
            'set-cookie': 'must-not-pass=1',
          },
        }
      )
    );

    const result = await client.request({
      method: 'GET',
      path: 'organizations',
      incomingHeaders: new Headers(),
    });

    expect(result).toMatchObject({
      ok: true,
      data: {
        envelope: {
          data: [{ id: 42 }],
          meta: { total: 1 },
        },
        responseHeaders: {
          vary: 'Organization-Id',
          'x-ratelimit-remaining': '99',
        },
      },
    });
    if (result.ok) {
      expect(result.data.responseHeaders).not.toHaveProperty('set-cookie');
    }
  });

  it('rejects oversized upstream bodies before parsing JSON', async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ code: 0, data: 'a'.repeat(2_048) }), {
        headers: { 'content-type': 'application/json' },
      })
    );

    await expect(
      client.request({
        method: 'GET',
        path: 'organizations',
        incomingHeaders: new Headers({ 'x-request-id': 'req-large-1' }),
      })
    ).resolves.toMatchObject({
      ok: false,
      error: {
        cause: 'unavailable',
        status: 503,
        errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
        requestId: 'req-large-1',
      },
    });
  });

  it('rejects absolute, traversal, query-bearing, and fragment-bearing paths', async () => {
    for (const path of [
      'https://evil.example/path',
      '../admin',
      '/organizations',
      'organizations?page=1',
      'organizations#fragment',
    ]) {
      expect(() =>
        client.request({
          method: 'GET',
          path,
          incomingHeaders: new Headers(),
        })
      ).toThrow('relative API path');
    }
    expect(fetchMock).not.toHaveBeenCalled();
  });
});

function successResponse(data: unknown): Response {
  return Response.json({ code: 0, message: 'success', data });
}
