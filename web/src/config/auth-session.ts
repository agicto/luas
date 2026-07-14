import 'server-only';

import { env } from '@/config/env';

type NodeEnvironment = typeof env.NODE_ENV;

export const AUTH_SESSION_MAX_AGE_SECONDS = 60 * 60 * 24 * 30;

export function getAuthSessionCookieName(
  environment: NodeEnvironment = env.NODE_ENV
): string {
  return environment === 'production' ? '__Host-luas_auth' : 'luas_auth';
}

export function createAuthSessionCookie(
  token: string,
  maxAgeSeconds: number,
  environment: NodeEnvironment = env.NODE_ENV
) {
  const maxAge = Math.max(
    1,
    Math.min(Math.floor(maxAgeSeconds), AUTH_SESSION_MAX_AGE_SECONDS)
  );

  return {
    name: getAuthSessionCookieName(environment),
    value: token,
    httpOnly: true,
    sameSite: 'lax' as const,
    secure: environment === 'production',
    path: '/',
    maxAge,
    priority: 'high' as const,
  };
}

export function createExpiredAuthSessionCookie(
  environment: NodeEnvironment = env.NODE_ENV
) {
  return {
    ...createAuthSessionCookie('', 1, environment),
    maxAge: 0,
    expires: new Date(0),
  };
}
