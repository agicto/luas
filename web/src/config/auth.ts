/** Browser-safe auth navigation and route ownership. */
export const authConfig = {
  routes: {
    login: '/login',
    register: '/register',
    afterLogin: '/console',
    afterLogout: '/login',
  },

  protectedRoutes: ['/console', '/styleguide', '/i18n-test'] as const,
  publicOnlyRoutes: ['/login', '/register'] as const,
} as const;
