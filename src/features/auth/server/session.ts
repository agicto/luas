import { cookies } from 'next/headers';
import { authConfig, deserializeAuthSession, serializeAuthSession } from '@/config/auth';
import { isProd } from '@/config/env';
import type { AuthUser } from '@/features/auth/types';

export async function getSessionUser(): Promise<AuthUser | null> {
  const cookieStore = await cookies();
  return deserializeAuthSession(cookieStore.get(authConfig.cookies.session)?.value);
}

export async function setSessionCookie(user: AuthUser): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.set({
    name: authConfig.cookies.session,
    value: serializeAuthSession(user),
    httpOnly: true,
    sameSite: 'lax',
    secure: isProd,
    path: '/',
    maxAge: authConfig.sessionMaxAge,
  });
}

export async function clearSessionCookie(): Promise<void> {
  const cookieStore = await cookies();
  cookieStore.delete(authConfig.cookies.session);
}
