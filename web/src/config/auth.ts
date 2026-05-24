/**
 * Mock auth configuration shared by client, middleware, and route handlers.
 *
 * The (de)serialization helpers that used to live here have been replaced
 * by an HMAC-signed scheme in `@/lib/session-signing` and the server-only
 * `@/features/auth/server/session` helpers. This file is now just config.
 */
export const authConfig = {
  sessionMaxAge: 60 * 60 * 24 * 7,

  cookies: {
    session: 'luas_session',
  },

  /**
   * Demo credentials accepted by the mock /api/auth/login route handler.
   * Delete this block when wiring up a real backend.
   */
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
