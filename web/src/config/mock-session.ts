import 'server-only';

import { env } from '@/config/env';

type NodeEnvironment = typeof env.NODE_ENV;

export const MOCK_SESSION_MAX_AGE_SECONDS = 60 * 60 * 24 * 7;

export function getMockSessionCookieName(
  environment: NodeEnvironment = env.NODE_ENV
): string {
  return environment === 'production' ? '__Host-luas_session' : 'luas_session';
}

export function createMockSessionCookie(
  value: string,
  environment: NodeEnvironment = env.NODE_ENV
) {
  return {
    name: getMockSessionCookieName(environment),
    value,
    httpOnly: true,
    sameSite: 'lax' as const,
    secure: environment === 'production',
    path: '/',
    maxAge: MOCK_SESSION_MAX_AGE_SECONDS,
    priority: 'high' as const,
  };
}

export function createExpiredMockSessionCookie(
  environment: NodeEnvironment = env.NODE_ENV
) {
  return {
    ...createMockSessionCookie('', environment),
    maxAge: 0,
    expires: new Date(0),
  };
}
