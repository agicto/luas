import { describe, expect, it } from 'vitest';

import {
  hasAuthFieldError,
  isUnauthenticatedAuthError,
  resolveAuthErrorKey,
} from '@/features/auth/utils/auth-error';
import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

describe('auth error presentation', () => {
  it('maps stable login and registration errors to local copy', () => {
    expect(
      resolveAuthErrorKey(
        new ApiError('raw credential message', ApiErrorCode.AUTH_INVALID_CREDENTIALS, 401),
        'login'
      )
    ).toBe('auth.invalidCredentials');

    expect(
      resolveAuthErrorKey(
        new ApiError('raw duplicate message', ApiErrorCode.USER_EMAIL_ALREADY_EXISTS, 409),
        'register'
      )
    ).toBe('auth.emailAlreadyExists');
  });

  it('prefers canonical error codes over conflicting HTTP status fallbacks', () => {
    expect(
      resolveAuthErrorKey(
        new ApiError(
          'raw disabled message',
          ApiErrorCode.AUTH_ACCOUNT_DISABLED,
          401
        ),
        'login'
      )
    ).toBe('auth.accountDisabled');
  });

  it('maps availability and malformed-response failures without exposing raw messages', () => {
    expect(
      resolveAuthErrorKey(new ApiError('raw timeout message', ClientErrorCode.TIMEOUT), 'login')
    ).toBe('errors.authUnavailable');

    expect(
      resolveAuthErrorKey(
        new ApiError('raw decoder message', ClientErrorCode.INVALID_RESPONSE),
        'register'
      )
    ).toBe('errors.authUnavailable');
  });

  it('detects field ownership without displaying backend field copy', () => {
    const error = new ApiError(
      'raw validation message',
      ApiErrorCode.COMMON_VALIDATION_FAILED,
      422,
      'request-1',
      {
        email: ['backend email detail'],
        password: ['backend password detail'],
      }
    );

    expect(resolveAuthErrorKey(error, 'register')).toBe('errors.validationFailed');
    expect(hasAuthFieldError(error, 'email')).toBe(true);
    expect(hasAuthFieldError(error, 'password')).toBe(true);
    expect(hasAuthFieldError(error, 'name')).toBe(false);
  });

  it('recognizes an already-absent logout session', () => {
    expect(
      isUnauthenticatedAuthError(
        new ApiError('raw unauthorized message', ApiErrorCode.AUTH_UNAUTHORIZED)
      )
    ).toBe(true);
    expect(
      isUnauthenticatedAuthError(
        new ApiError('raw status message', ClientErrorCode.FETCH_ERROR, 401)
      )
    ).toBe(true);
    expect(
      isUnauthenticatedAuthError(
        new ApiError('raw unavailable message', ClientErrorCode.NETWORK_ERROR)
      )
    ).toBe(false);
    expect(
      isUnauthenticatedAuthError(
        new ApiError(
          'raw disabled message',
          ApiErrorCode.AUTH_ACCOUNT_DISABLED,
          401
        )
      )
    ).toBe(false);
  });
});
