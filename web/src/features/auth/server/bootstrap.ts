import 'server-only';

import type { AuthBootstrap } from '@/features/auth/types';
import { getAuthRuntimeMode } from './auth-runtime';
import { getSessionUser } from './session';

export async function resolveAuthBootstrap(): Promise<AuthBootstrap> {
  if (getAuthRuntimeMode() === 'client-session') {
    return { status: 'client-required' };
  }

  const user = await getSessionUser();

  return user
    ? { status: 'authenticated', user }
    : { status: 'unauthenticated' };
}
