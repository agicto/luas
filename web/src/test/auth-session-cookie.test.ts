import { describe, expect, it } from 'vitest';

import {
  AUTH_SESSION_MAX_AGE_SECONDS,
  createAuthSessionCookie,
  createExpiredAuthSessionCookie,
  getAuthSessionCookieName,
} from '@/config/auth-session';
import {
  authTokenMaxAgeSeconds,
  isCompactJwt,
} from '@/features/auth/server/auth-token';

describe('production auth session cookie', () => {
  it('uses a host-only secure cookie in production', () => {
    expect(getAuthSessionCookieName('production')).toBe('__Host-luas_auth');
    expect(createAuthSessionCookie('a.b.c', 600, 'production')).toEqual({
      name: '__Host-luas_auth',
      value: 'a.b.c',
      httpOnly: true,
      sameSite: 'lax',
      secure: true,
      path: '/',
      maxAge: 600,
      priority: 'high',
    });
  });

  it('uses a separate non-secure name for local HTTP development', () => {
    expect(getAuthSessionCookieName('development')).toBe('luas_auth');
    expect(createAuthSessionCookie('a.b.c', 600, 'development')).toMatchObject({
      name: 'luas_auth',
      secure: false,
    });
  });

  it('caps token persistence and expires the exact cookie scope', () => {
    expect(
      createAuthSessionCookie(
        'a.b.c',
        AUTH_SESSION_MAX_AGE_SECONDS * 2,
        'production'
      ).maxAge
    ).toBe(AUTH_SESSION_MAX_AGE_SECONDS);

    expect(createExpiredAuthSessionCookie('production')).toMatchObject({
      name: '__Host-luas_auth',
      value: '',
      maxAge: 0,
      expires: new Date(0),
      path: '/',
    });
  });

  it('accepts only bounded compact JWT values', () => {
    expect(isCompactJwt('header.payload.signature')).toBe(true);
    expect(isCompactJwt('not-a-jwt')).toBe(false);
    expect(isCompactJwt(`a.${'b'.repeat(3_500)}.c`)).toBe(false);
  });

  it('derives cookie lifetime only from a valid future integer exp claim', () => {
    const token = compactJwt({ exp: 1_700_000_600 });

    expect(authTokenMaxAgeSeconds(token, 1_700_000_000_000)).toBe(600);
    expect(
      authTokenMaxAgeSeconds(compactJwt({ exp: 1_699_999_999 }), 1_700_000_000_000)
    ).toBeNull();
    expect(authTokenMaxAgeSeconds(compactJwt({ exp: 'soon' }))).toBeNull();
  });
});

function compactJwt(payload: Record<string, unknown>): string {
  return [
    Buffer.from(JSON.stringify({ alg: 'HS256' })).toString('base64url'),
    Buffer.from(JSON.stringify(payload)).toString('base64url'),
    'signature',
  ].join('.');
}
