import 'server-only';

import { cookies } from 'next/headers';

import {
  createAuthSessionCookie,
  createExpiredAuthSessionCookie,
  getAuthSessionCookieName,
} from '@/config/auth-session';
import { isOpaqueAuthenticationCredential } from '@/features/auth/server/auth-credential';

export async function getApiSessionToken(): Promise<string | null> {
  const cookieStore = await cookies();
  const token = cookieStore.get(getAuthSessionCookieName())?.value;

  return isOpaqueAuthenticationCredential(token) ? token : null;
}

export async function setApiSessionCookie(
  token: string,
  maxAgeSeconds: number
): Promise<void> {
  if (!isOpaqueAuthenticationCredential(token)) {
    throw new Error('Refusing to store a malformed API access token');
  }

  const cookieStore = await cookies();
  cookieStore.set(createAuthSessionCookie(token, maxAgeSeconds));
}

export async function clearApiSessionCookie(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.set(createExpiredAuthSessionCookie());
}
