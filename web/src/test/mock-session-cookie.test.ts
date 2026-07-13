import { describe, expect, it } from 'vitest';

import {
  createExpiredMockSessionCookie,
  createMockSessionCookie,
  getMockSessionCookieName,
} from '@/config/mock-session';

describe('mock session cookie policy', () => {
  it('uses a __Host- cookie with secure production attributes', () => {
    expect(getMockSessionCookieName('production')).toBe('__Host-luas_session');
    expect(createMockSessionCookie('signed-value', 'production')).toMatchObject({
      name: '__Host-luas_session',
      value: 'signed-value',
      httpOnly: true,
      sameSite: 'lax',
      secure: true,
      path: '/',
      maxAge: 60 * 60 * 24 * 7,
      priority: 'high',
    });
  });

  it('keeps a non-prefixed cookie for local HTTP development', () => {
    expect(getMockSessionCookieName('development')).toBe('luas_session');
    expect(createMockSessionCookie('signed-value', 'development')).toMatchObject({
      name: 'luas_session',
      secure: false,
    });
  });

  it('clears the exact cookie scope and security attributes', () => {
    const expired = createExpiredMockSessionCookie('production');

    expect(expired).toMatchObject({
      name: '__Host-luas_session',
      value: '',
      httpOnly: true,
      sameSite: 'lax',
      secure: true,
      path: '/',
      maxAge: 0,
      priority: 'high',
    });
    expect(expired.expires).toEqual(new Date(0));
  });
});
