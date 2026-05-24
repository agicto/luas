import type { AuthUser } from '@/features/auth/types';

/**
 * Mock auth configuration shared by client, middleware, and route handlers.
 */
export const authConfig = {
  sessionMaxAge: 60 * 60 * 24 * 7,

  cookies: {
    session: 'luas_session',
  },

  demoUser: {
    id: 'demo-admin',
    email: 'admin@example.com',
    password: 'admin123',
    name: 'Admin User',
    role: 'admin',
  } as const,

  routes: {
    login: '/login',
    register: '/register',
    afterLogin: '/console',
    afterLogout: '/login',
  },

  protectedRoutes: ['/console', '/styleguide', '/i18n-test'] as const,
  publicOnlyRoutes: ['/login', '/register'] as const,
} as const;

export function serializeAuthSession(user: AuthUser): string {
  return encodeURIComponent(JSON.stringify(user));
}

export function deserializeAuthSession(value?: string | null): AuthUser | null {
  if (!value) {
    return null;
  }

  try {
    const parsed = JSON.parse(decodeURIComponent(value)) as AuthUser;

    if (
      typeof parsed.id !== 'string' ||
      typeof parsed.email !== 'string' ||
      typeof parsed.name !== 'string' ||
      typeof parsed.role !== 'string'
    ) {
      return null;
    }

    return parsed;
  } catch {
    return null;
  }
}
