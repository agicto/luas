import { describe, expect, it } from 'vitest';

import { POST as login } from '@/app/api/auth/login/route';
import { GET as getExample } from '@/app/api/example/[id]/route';
import { ApiErrorCode } from '@/http/codes';

describe('mock API error contract', () => {
  it('returns 400 COMMON.INVALID_INPUT for malformed JSON bodies', async () => {
    const response = await login(
      new Request('http://localhost/api/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: '{',
      })
    );

    await expect(response.json()).resolves.toEqual({
      code: 400,
      error_code: ApiErrorCode.COMMON_INVALID_INPUT,
      message: 'Malformed JSON body',
    });
    expect(response.status).toBe(400);
  });

  it('returns 422 COMMON.VALIDATION_FAILED with field errors for schema failures', async () => {
    const response = await login(
      new Request('http://localhost/api/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          email: 'not-an-email',
          password: '',
        }),
      })
    );
    const body = await response.json();

    expect(response.status).toBe(422);
    expect(body).toMatchObject({
      code: 422,
      error_code: ApiErrorCode.COMMON_VALIDATION_FAILED,
      message: 'Invalid login payload',
    });
    expect(body.errors.email).toEqual(expect.arrayContaining([expect.any(String)]));
    expect(body.errors.password).toEqual(expect.arrayContaining([expect.any(String)]));
  });

  it('returns 404 COMMON.NOT_FOUND for missing mock resources', async () => {
    const response = await getExample(new Request('http://localhost/api/example/missing-id'), {
      params: Promise.resolve({
        id: 'missing-id',
      }),
    });

    await expect(response.json()).resolves.toEqual({
      code: 404,
      error_code: ApiErrorCode.COMMON_NOT_FOUND,
      message: 'Example item not found',
    });
    expect(response.status).toBe(404);
  });
});
