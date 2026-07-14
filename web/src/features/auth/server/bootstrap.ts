import 'server-only';

import type { AuthBootstrap } from '@/features/auth/types';
import { resolveGoApiAuthBootstrap } from './auth-adapter-route';
import { getAuthRuntimeMode } from './auth-runtime';
import { getSessionUser } from './session';

export async function resolveAuthBootstrap(): Promise<AuthBootstrap> {
  const mode = getAuthRuntimeMode();

  if (mode === 'client-session') {
    return { status: 'client-required' };
  }
  if (mode === 'api-session') {
    return resolveGoApiAuthBootstrap();
  }

  const user = await getSessionUser();

  return user
    ? { status: 'authenticated', user }
    : { status: 'unauthenticated' };
}
