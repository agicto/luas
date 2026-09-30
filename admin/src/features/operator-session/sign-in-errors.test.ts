import { ApiError } from '@/http/client';
import { ClientErrorCode } from '@/http/codes';
import { isSignedOut } from '@/features/operator-session/hooks/use-operator-session';
import { signInErrorKey } from '@/features/operator-session/sign-in-errors';

describe('operator sign-in errors', () => {
  it.each([
    ['AUTH.INVALID_CREDENTIALS', 'auth.errors.invalidCredentials'],
    ['OPERATOR.FORBIDDEN', 'auth.errors.notOperator'],
    ['COMMON.RATE_LIMITED', 'auth.errors.rateLimited'],
    ['OPERATOR.ORIGIN_REJECTED', 'auth.errors.originRejected'],
    ['COMMON.SERVICE_UNAVAILABLE', 'auth.errors.unavailable'],
  ])('maps %s to %s', (code, key) => {
    expect(signInErrorKey(new ApiError('upstream text', code))).toBe(key);
  });

  it('never surfaces transport failures as credential errors', () => {
    expect(signInErrorKey(new ApiError('offline', ClientErrorCode.NETWORK_ERROR))).toBe(
      'auth.errors.unavailable',
    );
    expect(signInErrorKey(new Error('boom'))).toBe('auth.errors.unavailable');
  });

  it('treats unauthenticated, non-operator, and disabled sessions as signed out', () => {
    expect(isSignedOut(new ApiError('x', 'AUTH.UNAUTHORIZED', { status: 401 }))).toBe(true);
    expect(isSignedOut(new ApiError('x', 'OPERATOR.FORBIDDEN', { status: 403 }))).toBe(true);
    expect(isSignedOut(new ApiError('x', 'AUTH.ACCOUNT_DISABLED', { status: 403 }))).toBe(true);
    expect(isSignedOut(new ApiError('x', 'COMMON.SERVICE_UNAVAILABLE', { status: 503 }))).toBe(
      false,
    );
  });
});
