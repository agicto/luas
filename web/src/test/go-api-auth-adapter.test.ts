import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  generatedUsername,
  GoApiAuthAdapter,
  mapGoApiUser,
} from '@/features/auth/server/go-api-auth-adapter';
import { ApiErrorCode } from '@/http/codes';

const now = 1_700_000_000_000;
const randomId = '01234567-89ab-cdef-0123-456789abcdef';
const activeApiUser = {
  id: 42,
  username: 'ada',
  email: 'ada@example.com',
  nickname: 'Ada Lovelace',
  status: 1,
};

describe('GoApiAuthAdapter', () => {
  let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;
  let adapter: GoApiAuthAdapter;

  beforeEach(() => {
    fetchMock = vi.fn<typeof fetch>();
    adapter = new GoApiAuthAdapter(
      {
        apiUrl: 'https://api.example.com/v1',
        timeoutMs: 5_000,
        clientIpHeader: 'x-real-client-ip',
      },
      {
        fetch: fetchMock,
        now: () => now,
        randomUUID: () => randomId,
      }
    );
  });

  it('maps browser login to the fixed Go endpoint and returns a bounded session', async () => {
    fetchMock.mockResolvedValueOnce(
      successResponse({
        access_token: compactJwt({ exp: 1_700_000_600 }),
        user: activeApiUser,
      })
    );
    const incoming = new Headers({
      authorization: 'Bearer browser-controlled',
      cookie: 'browser=controlled',
      'x-real-client-ip': '203.0.113.9',
      'x-request-id': 'req-login-1',
    });

    await expect(
      adapter.login(
        { email: 'ada@example.com', password: 'secret-123' },
        incoming
      )
    ).resolves.toEqual({
      ok: true,
      data: {
        user: {
          id: '42',
          email: 'ada@example.com',
          name: 'Ada Lovelace',
        },
        accessToken: compactJwt({ exp: 1_700_000_600 }),
        maxAgeSeconds: 600,
      },
    });

    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe('https://api.example.com/v1/login');
    expect(init).toMatchObject({
      method: 'POST',
      cache: 'no-store',
      redirect: 'error',
      body: JSON.stringify({
        username: 'ada@example.com',
        password: 'secret-123',
      }),
    });
    const headers = new Headers(init?.headers);
    expect(headers.get('x-forwarded-for')).toBe('203.0.113.9');
    expect(headers.get('x-request-id')).toBe('req-login-1');
    expect(headers.get('authorization')).toBeNull();
    expect(headers.get('cookie')).toBeNull();
  });

  it('registers with a non-identifying generated username, then logs in by email', async () => {
    fetchMock
      .mockResolvedValueOnce(successResponse(activeApiUser, 201))
      .mockResolvedValueOnce(
        successResponse({
          access_token: compactJwt({ exp: 1_700_000_900 }),
          user: activeApiUser,
        })
      );

    const result = await adapter.register(
      {
        name: 'Ada Lovelace',
        email: 'ada@example.com',
        password: 'secret-123',
      },
      new Headers({ 'x-request-id': 'req-register-1' })
    );

    expect(result).toMatchObject({
      ok: true,
      data: {
        user: { id: '42', name: 'Ada Lovelace' },
        maxAgeSeconds: 900,
      },
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);

    const [registerUrl, registerInit] = fetchMock.mock.calls[0];
    expect(String(registerUrl)).toBe('https://api.example.com/v1/register');
    expect(JSON.parse(String(registerInit?.body))).toEqual({
      username: 'user_0123456789abcdef0123456789abcdef',
      password: 'secret-123',
      email: 'ada@example.com',
      nickname: 'Ada Lovelace',
    });

    const [loginUrl, loginInit] = fetchMock.mock.calls[1];
    expect(String(loginUrl)).toBe('https://api.example.com/v1/login');
    expect(JSON.parse(String(loginInit?.body))).toEqual({
      username: 'ada@example.com',
      password: 'secret-123',
    });
  });

  it('loads the profile with only the server-owned bearer token', async () => {
    fetchMock.mockResolvedValueOnce(successResponse(activeApiUser));
    const token = compactJwt({ exp: 1_700_000_600 });

    await expect(
      adapter.profile(token, new Headers({ cookie: 'untrusted=1' }))
    ).resolves.toEqual({
      ok: true,
      data: {
        user: {
          id: '42',
          email: 'ada@example.com',
          name: 'Ada Lovelace',
        },
      },
    });

    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe('https://api.example.com/v1/users/profile');
    const headers = new Headers(init?.headers);
    expect(headers.get('authorization')).toBe(`Bearer ${token}`);
    expect(headers.get('cookie')).toBeNull();
  });

  it('preserves canonical errors and field ownership without backend copy', async () => {
    fetchMock.mockResolvedValueOnce(
      errorResponse(
        422,
        ApiErrorCode.COMMON_VALIDATION_FAILED,
        {
          username: ['generated username leaked'],
          nickname: ['raw backend name detail'],
          email: ['raw backend email detail'],
        },
        {
          'retry-after': '30',
          'x-request-id': 'req-upstream-422',
        }
      )
    );

    await expect(
      adapter.register(
        {
          name: 'Ada',
          email: 'ada@example.com',
          password: 'secret-123',
        },
        new Headers()
      )
    ).resolves.toEqual({
      ok: false,
      error: {
        status: 422,
        errorCode: ApiErrorCode.COMMON_VALIDATION_FAILED,
        message: 'Invalid authentication input',
        fieldErrors: {
          name: ['Invalid value'],
          email: ['Invalid value'],
        },
        requestId: 'req-upstream-422',
        responseHeaders: { 'retry-after': '30' },
      },
    });
  });

  it('classifies disabled users without inventing a browser role', async () => {
    fetchMock.mockResolvedValueOnce(
      successResponse({ ...activeApiUser, status: 0 })
    );

    await expect(
      adapter.profile(compactJwt({ exp: 1_700_000_600 }), new Headers())
    ).resolves.toMatchObject({
      ok: false,
      error: {
        status: 403,
        errorCode: ApiErrorCode.AUTH_ACCOUNT_DISABLED,
      },
    });
  });

  it('turns malformed success data and expired tokens into availability failures', async () => {
    fetchMock
      .mockResolvedValueOnce(successResponse({ user: activeApiUser }))
      .mockResolvedValueOnce(
        successResponse({
          access_token: compactJwt({ exp: 1_699_999_999 }),
          user: activeApiUser,
        })
      );

    await expect(
      adapter.login(
        { email: 'ada@example.com', password: 'secret-123' },
        new Headers({ 'x-request-id': 'req-malformed-1' })
      )
    ).resolves.toMatchObject({
      ok: false,
      error: {
        status: 503,
        errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      },
    });

    await expect(
      adapter.login(
        { email: 'ada@example.com', password: 'secret-123' },
        new Headers({ 'x-request-id': 'req-expired-1' })
      )
    ).resolves.toMatchObject({
      ok: false,
      error: {
        status: 503,
        errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      },
    });
  });

  it('maps aborts to the canonical timeout and rejects forwarded IP chains', async () => {
    fetchMock.mockRejectedValueOnce(new DOMException('timed out', 'TimeoutError'));

    const result = await adapter.login(
      { email: 'ada@example.com', password: 'secret-123' },
      new Headers({
        'x-real-client-ip': '198.51.100.99, 203.0.113.9',
        'x-request-id': 'contains spaces',
      })
    );

    expect(result).toMatchObject({
      ok: false,
      error: {
        status: 503,
        errorCode: ApiErrorCode.COMMON_TIMEOUT,
        requestId: randomId,
      },
    });
    const headers = new Headers(fetchMock.mock.calls[0][1]?.headers);
    expect(headers.get('x-forwarded-for')).toBeNull();
    expect(headers.get('x-request-id')).toBe(randomId);
  });
});

describe('Go API auth mapping helpers', () => {
  it('keeps generated usernames inside the API DTO limit', () => {
    const username = generatedUsername(randomId);

    expect(username).toBe('user_0123456789abcdef0123456789abcdef');
    expect(username.length).toBeLessThanOrEqual(50);
  });

  it('uses nickname first and never adds unproven authorization fields', () => {
    expect(mapGoApiUser(activeApiUser)).toEqual({
      id: '42',
      email: 'ada@example.com',
      name: 'Ada Lovelace',
    });
    expect(
      mapGoApiUser({ ...activeApiUser, nickname: '  ', username: 'ada' })
    ).toEqual({
      id: '42',
      email: 'ada@example.com',
      name: 'ada',
    });
  });
});

function successResponse(data: unknown, status = 200): Response {
  return Response.json({ code: 0, message: 'success', data }, { status });
}

function errorResponse(
  status: number,
  errorCode: string,
  errors?: Record<string, string[]>,
  headers?: HeadersInit
): Response {
  return Response.json(
    {
      code: status,
      error_code: errorCode,
      message: 'raw backend detail must not escape',
      ...(errors ? { errors } : {}),
      request_id: new Headers(headers).get('x-request-id'),
    },
    { status, headers }
  );
}

function compactJwt(payload: Record<string, unknown>): string {
  return [
    Buffer.from(JSON.stringify({ alg: 'HS256' })).toString('base64url'),
    Buffer.from(JSON.stringify(payload)).toString('base64url'),
    'signature',
  ].join('.');
}
