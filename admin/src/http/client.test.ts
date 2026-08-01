import { z } from 'zod';
import { ClientErrorCode } from '@/http/codes';
import { ApiError, http } from '@/http/client';

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
});
