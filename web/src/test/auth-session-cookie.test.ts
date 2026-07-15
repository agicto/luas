import { describe, expect, it } from 'vitest';

import {
  AUTH_SESSION_MAX_AGE_SECONDS,
  createAuthSessionCookie,
  createExpiredAuthSessionCookie,
  getAuthSessionCookieName,
} from '@/config/auth-session';
import {
  authenticationSessionMaxAgeSeconds,
  isOpaqueAuthenticationCredential,
} from '@/features/auth/server/auth-credential';

const opaqueCredential = Buffer.from(
  '0123456789abcdef0123456789abcdef'
).toString('base64url');

describe('production auth session cookie', () => {
  it('uses a host-only secure cookie in production', () => {
    expect(getAuthSessionCookieName('production')).toBe('__Host-luas_auth');
    expect(createAuthSessionCookie(opaqueCredential, 600, 'production')).toEqual({
      name: '__Host-luas_auth',
      value: opaqueCredential,
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
    expect(createAuthSessionCookie(opaqueCredential, 600, 'development')).toMatchObject({
      name: 'luas_auth',
      secure: false,
    });
  });

  it('caps token persistence and expires the exact cookie scope', () => {
    expect(
      createAuthSessionCookie(
        opaqueCredential,
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

  it('accepts only exact 256-bit base64url credentials', () => {
    expect(isOpaqueAuthenticationCredential(opaqueCredential)).toBe(true);
    expect(isOpaqueAuthenticationCredential('header.payload.signature')).toBe(false);
    expect(isOpaqueAuthenticationCredential('a'.repeat(42))).toBe(false);
    expect(isOpaqueAuthenticationCredential(`${'a'.repeat(42)}!`)).toBe(false);
  });

  it('accepts only a positive safe expires_in and caps browser persistence', () => {
    expect(authenticationSessionMaxAgeSeconds(600)).toBe(600);
    expect(authenticationSessionMaxAgeSeconds(AUTH_SESSION_MAX_AGE_SECONDS * 2)).toBe(
      AUTH_SESSION_MAX_AGE_SECONDS
    );
    expect(authenticationSessionMaxAgeSeconds(0)).toBeNull();
    expect(authenticationSessionMaxAgeSeconds('soon')).toBeNull();
  });
});
