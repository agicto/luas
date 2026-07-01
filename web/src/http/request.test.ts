import { describe, expect, it } from 'vitest';
import type { AxiosError } from 'axios';

import { ApiErrorCode, ClientErrorCode } from './codes';
import { toApiError } from './request';

describe('toApiError', () => {
  it('prefers Go API error_code over numeric status code', () => {
    const error = {
      message: 'Request failed',
      response: {
        status: 409,
        data: {
          code: 409,
          error_code: 'USER.EMAIL_ALREADY_EXISTS',
          errors: {
            email: ['Email already exists'],
          },
          message: 'Registration failed',
          request_id: 'req_123',
        },
      },
    } as AxiosError;

    const apiError = toApiError(error);

    expect(apiError.message).toBe('Registration failed');
    expect(apiError.errorCode).toBe(ApiErrorCode.USER_EMAIL_ALREADY_EXISTS);
    expect(apiError.fieldErrors).toEqual({
      email: ['Email already exists'],
    });
    expect(apiError.status).toBe(409);
    expect(apiError.requestId).toBe('req_123');
  });

  it('falls back to canonical API codes when no body error_code exists', () => {
    const error = {
      message: 'Request failed',
      response: {
        status: 404,
        data: {},
      },
    } as AxiosError;

    expect(toApiError(error).errorCode).toBe(ApiErrorCode.COMMON_NOT_FOUND);
  });

  it('normalizes legacy frontend string codes before status fallback', () => {
    const error = {
      message: 'Request failed',
      response: {
        status: 401,
        data: {
          error: 'Invalid email or password',
          code: 'AUTH_002',
        },
      },
    } as AxiosError;

    const apiError = toApiError(error);

    expect(apiError.message).toBe('Invalid email or password');
    expect(apiError.errorCode).toBe(ApiErrorCode.AUTH_INVALID_CREDENTIALS);
  });

  it('uses client-scoped codes for transport failures without a response', () => {
    const error = {
      message: 'Network Error',
      code: 'ERR_NETWORK',
    } as AxiosError;

    expect(toApiError(error).errorCode).toBe(ClientErrorCode.NETWORK_ERROR);
  });
});
