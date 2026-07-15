import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const cookieStore = vi.hoisted(() => ({
  get: vi.fn(),
  set: vi.fn(),
}));
const requestHeaders = vi.hoisted(() => new Headers({
  'x-forwarded-for': '203.0.113.20',
  'x-request-id': 'req-bootstrap-1',
}));

vi.mock('next/headers', () => ({
  cookies: async () => cookieStore,
  headers: async () => requestHeaders,
}));

const originalEnv = { ...process.env };

describe('production auth adapter route boundary', () => {
  let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;

  beforeEach(() => {
    vi.resetModules();
    fetchMock = vi.fn<typeof fetch>();
    vi.stubGlobal('fetch', fetchMock);
    cookieStore.get.mockReset();
    cookieStore.set.mockReset();

    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_TIMEOUT_MS = '5000';
    process.env.API_UPSTREAM_MAX_RESPONSE_BYTES = '1048576';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    process.env.API_CLIENT_IP_HEADER = 'x-forwarded-for';
    process.env.NEXT_PUBLIC_API_URL = '/api';
    process.env.NEXT_PUBLIC_APP_URL = 'https://app.example.com';
    delete process.env.NEXT_PHASE;
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
    vi.resetModules();

    for (const key of [
      'API_ADAPTER_ENABLED',
      'API_UPSTREAM_TIMEOUT_MS',
      'API_UPSTREAM_MAX_RESPONSE_BYTES',
      'API_UPSTREAM_URL',
      'API_CLIENT_IP_HEADER',
      'NEXT_PHASE',
      'NEXT_PUBLIC_API_URL',
      'NEXT_PUBLIC_APP_URL',
    ] as const) {
      delete process.env[key];
      if (originalEnv[key] !== undefined) {
        process.env[key] = originalEnv[key];
      }
    }
  });

  it('sets the HttpOnly API cookie without exposing the bearer token', async () => {
    const token = compactJwt(Math.floor(Date.now() / 1_000) + 600);
    fetchMock.mockResolvedValueOnce(loginResponse(token));
    const { loginWithGoApi } = await import(
      '@/features/auth/server/auth-adapter-route'
    );

    const response = await loginWithGoApi(
      authRequest('/api/auth/login'),
      { email: 'ada@example.com', password: 'secret-123' }
    );

    expect(response.status).toBe(200);
    const body = await response.json();
    expect(body).toEqual({
      code: 0,
      message: 'success',
      data: {
        user: {
          id: '42',
          email: 'ada@example.com',
          name: 'Ada Lovelace',
        },
      },
    });
    expect(JSON.stringify(body)).not.toContain(token);
    expect(cookieStore.set).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'luas_session',
        maxAge: 0,
      })
    );
    expect(cookieStore.set).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'luas_auth',
        value: token,
        httpOnly: true,
        maxAge: expect.any(Number),
      })
    );
  });

  it('clears a rejected API session and preserves the canonical upstream error', async () => {
    const token = compactJwt(Math.floor(Date.now() / 1_000) + 600);
    cookieStore.get.mockReturnValue({ value: token });
    fetchMock.mockResolvedValueOnce(
      Response.json(
        {
          code: 401,
          error_code: 'AUTH.UNAUTHORIZED',
          message: 'raw upstream detail',
          request_id: 'req-upstream-401',
        },
        { status: 401 }
      )
    );
    const { getCurrentGoApiSession } = await import(
      '@/features/auth/server/auth-adapter-route'
    );

    const response = await getCurrentGoApiSession(
      authRequest('/api/auth/me', 'GET')
    );

    expect(response.status).toBe(401);
    await expect(response.json()).resolves.toEqual({
      code: 401,
      error_code: 'AUTH.UNAUTHORIZED',
      message: 'Authentication required',
      request_id: 'req-upstream-401',
    });
    expect(cookieStore.set).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'luas_auth',
        maxAge: 0,
      })
    );
  });

  it('expires both auth schemes on logout without calling an absent revoke endpoint', async () => {
    const { logoutFromGoApi } = await import(
      '@/features/auth/server/auth-adapter-route'
    );

    const response = await logoutFromGoApi();

    expect(response.status).toBe(200);
    await expect(response.json()).resolves.toMatchObject({
      data: { success: true },
    });
    expect(fetchMock).not.toHaveBeenCalled();
    expect(cookieStore.set.mock.calls.map(([cookie]) => cookie.name).sort()).toEqual([
      'luas_auth',
      'luas_session',
    ]);
  });

  it('keeps an upstream outage distinct from an absent server session', async () => {
    cookieStore.get.mockReturnValue({
      value: compactJwt(Math.floor(Date.now() / 1_000) + 600),
    });
    fetchMock.mockRejectedValueOnce(new Error('upstream offline'));
    const { resolveGoApiAuthBootstrap } = await import(
      '@/features/auth/server/auth-adapter-route'
    );

    await expect(resolveGoApiAuthBootstrap()).resolves.toEqual({
      status: 'unavailable',
    });
  });
});

function authRequest(path: string, method = 'POST'): Request {
  return new Request(`https://app.example.com${path}`, {
    method,
    headers: requestHeaders,
  });
}

function loginResponse(token: string): Response {
  return Response.json({
    code: 0,
    message: 'success',
    data: {
      access_token: token,
      user: {
        id: 42,
        username: 'ada',
        email: 'ada@example.com',
        nickname: 'Ada Lovelace',
        status: 1,
      },
    },
  });
}

function compactJwt(exp: number): string {
  return [
    Buffer.from(JSON.stringify({ alg: 'HS256' })).toString('base64url'),
    Buffer.from(JSON.stringify({ exp })).toString('base64url'),
    'signature',
  ].join('.');
}
