import { cookies } from 'next/headers';

import {
  createExpiredMockSessionCookie,
  createMockSessionCookie,
  getMockSessionCookieName,
  MOCK_SESSION_MAX_AGE_SECONDS,
} from '@/config/mock-session';
import { signSession, verifySession } from '@/lib/session-signing';
import type { AuthUser } from '@/features/auth/types';
import { isAuthUser } from '@/features/auth/utils/auth-user';

/**
 * Session helpers — MOCK SCHEME.
 *
 * The cookie value is `base64url(payload).base64url(hmac)` where the
 * payload is JSON of the user record plus an issued-at + expiry. The
 * server can't revoke individual sessions (no backend store) — that's
 * the trade-off of staying mock-only. Drop these in favor of opaque
 * tokens issued by your real backend before going to production.
 */

type SessionPayload = AuthUser & {
  iat: number;
  exp: number;
};

export async function getSessionUser(): Promise<AuthUser | null> {
  const cookieStore = await cookies();
  const raw = cookieStore.get(getMockSessionCookieName())?.value;
  return parseSession(await verifySession(raw));
}

export async function setSessionCookie(user: AuthUser): Promise<void> {
  const now = Math.floor(Date.now() / 1000);
  const payload: SessionPayload = {
    ...user,
    iat: now,
    exp: now + MOCK_SESSION_MAX_AGE_SECONDS,
  };
  const signed = await signSession(JSON.stringify(payload));

  const cookieStore = await cookies();
  cookieStore.set(createMockSessionCookie(signed));
}

export async function clearSessionCookie(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.set(createExpiredMockSessionCookie());
}

function parseSession(payload: string | null): AuthUser | null {
  if (!payload) return null;

  let parsed: unknown;
  try {
    parsed = JSON.parse(payload);
  } catch {
    return null;
  }

  if (!isSessionPayload(parsed)) return null;
  if (parsed.exp <= Math.floor(Date.now() / 1000)) return null;

  // Strip server-only fields before handing it back as AuthUser.
  const { iat: _iat, exp: _exp, ...user } = parsed;
  void _iat;
  void _exp;
  return user;
}

function isSessionPayload(value: unknown): value is SessionPayload {
  if (typeof value !== 'object' || value === null) return false;
  const v = value as Record<string, unknown>;
  return (
    isAuthUser(value) &&
    typeof v.iat === 'number' &&
    typeof v.exp === 'number'
  );
}
