import { AxiosError, type InternalAxiosRequestConfig } from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiErrorCode } from '@/http/codes';
import { createRequest } from '@/http/request';

const handleError = vi.hoisted(() => vi.fn());

vi.mock('@/http/error-handler', () => ({ handleError }));

function rejectingAdapter(config: InternalAxiosRequestConfig) {
  return Promise.reject(
    new AxiosError(
      'Request failed with status code 422',
      AxiosError.ERR_BAD_REQUEST,
      config,
      undefined,
      {
        config,
        data: {
          code: 422,
          error_code: ApiErrorCode.COMMON_VALIDATION_FAILED,
          errors: {
            email: ['backend email detail'],
          },
          message: 'backend validation detail',
          request_id: 'request-422',
        },
        headers: {},
        status: 422,
        statusText: 'Unprocessable Entity',
      }
    )
  );
}

describe('HTTP error handling ownership', () => {
  beforeEach(() => {
    handleError.mockReset();
  });

  it('normalizes errors without producing presentation side effects', async () => {
    const request = createRequest({ adapter: rejectingAdapter });

    await expect(request.post('/auth/register')).rejects.toMatchObject({
      errorCode: ApiErrorCode.COMMON_VALIDATION_FAILED,
      status: 422,
    });
    expect(handleError).not.toHaveBeenCalled();
  });

  it('preserves diagnostic metadata for the owning caller', async () => {
    const request = createRequest({ adapter: rejectingAdapter });

    await expect(request.post('/example')).rejects.toMatchObject({
      errorCode: ApiErrorCode.COMMON_VALIDATION_FAILED,
      fieldErrors: {
        email: ['backend email detail'],
      },
      requestId: 'request-422',
      status: 422,
    });
    expect(handleError).not.toHaveBeenCalled();
  });
});
