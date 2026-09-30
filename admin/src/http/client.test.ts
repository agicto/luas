import { z } from 'zod';
import { ClientErrorCode } from '@/http/codes';
import { ApiError, csrf, http } from '@/http/client';

describe('http client', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('extracts and validates a standard success envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: 0,
            message: 'success',
            data: { id: 'usr_1' },
          }),
          {
            headers: { 'content-type': 'application/json' },
            status: 200,
          },
        ),
      ),
    );

    await expect(
      http.get('/v1/users/profile', {
        schema: z.object({ id: z.string() }),
      }),
    ).resolves.toEqual({ id: 'usr_1' });
  });

  it('preserves canonical error_code and request_id', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            code: 401,
            error_code: 'AUTH.UNAUTHORIZED',
            message: 'Unauthorized',
            request_id: 'req_123',
          }),
          {
            headers: { 'content-type': 'application/json' },
            status: 401,
          },
        ),
      ),
    );

    const error = await http.get('/v1/users/profile').catch((value: unknown) => value);

    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({
      errorCode: 'AUTH.UNAUTHORIZED',
      requestId: 'req_123',
      status: 401,
    });
  });

  it('rejects a successful response that violates its schema', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ code: 0, message: 'success', data: { id: 42 } }), {
          headers: { 'content-type': 'application/json' },
          status: 200,
        }),
      ),
    );

    await expect(
      http.get('/v1/users/profile', {
        schema: z.object({ id: z.string() }),
      }),
    ).rejects.toMatchObject({
      errorCode: ClientErrorCode.INVALID_RESPONSE,
    });
  });

  describe('operator CSRF token', () => {
    afterEach(() => {
      csrf.set(undefined);
      csrf.onRejected(undefined);
    });

    const json = (body: unknown, status = 200) =>
      new Response(JSON.stringify(body), {
        headers: { 'content-type': 'application/json' },
        status,
      });

    it('adds the token to unsafe requests only', async () => {
      const fetchMock = vi
        .fn()
        .mockImplementation(() => Promise.resolve(json({ code: 0, message: 'ok', data: {} })));
      vi.stubGlobal('fetch', fetchMock);
      csrf.set('token-value-0123456789');

      await http.get('/v1/operator/session');
      await http.post('/v1/operator/users/1/disable');

      const getHeaders = new Headers((fetchMock.mock.calls[0]?.[1] as RequestInit).headers);
      const postHeaders = new Headers((fetchMock.mock.calls[1]?.[1] as RequestInit).headers);
      expect(getHeaders.has('X-CSRF-Token')).toBe(false);
      expect(postHeaders.get('X-CSRF-Token')).toBe('token-value-0123456789');
    });

    it('recovers once after a CSRF rejection and retries with the new token', async () => {
      const fetchMock = vi
        .fn()
        .mockResolvedValueOnce(
          json({ code: 403, error_code: 'OPERATOR.CSRF_REJECTED', message: 'rejected' }, 403),
        )
        .mockResolvedValueOnce(json({ code: 0, message: 'ok', data: { done: true } }));
      vi.stubGlobal('fetch', fetchMock);
      csrf.set('stale-token-0123456789');
      const recover = vi.fn().mockImplementation(() => {
        csrf.set('fresh-token-0123456789');
        return Promise.resolve();
      });
      csrf.onRejected(recover);

      await expect(http.post('/v1/operator/users/1/disable')).resolves.toEqual({ done: true });

      expect(recover).toHaveBeenCalledTimes(1);
      const retryHeaders = new Headers((fetchMock.mock.calls[1]?.[1] as RequestInit).headers);
      expect(retryHeaders.get('X-CSRF-Token')).toBe('fresh-token-0123456789');
    });

    it('does not retry a second CSRF rejection', async () => {
      const rejected = () =>
        json({ code: 403, error_code: 'OPERATOR.CSRF_REJECTED', message: 'rejected' }, 403);
      const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(rejected()));
      vi.stubGlobal('fetch', fetchMock);
      csrf.onRejected(() => Promise.resolve());

      await expect(http.post('/v1/operator/users/1/disable')).rejects.toMatchObject({
        errorCode: 'OPERATOR.CSRF_REJECTED',
      });
      expect(fetchMock).toHaveBeenCalledTimes(2);
    });
  });
});
