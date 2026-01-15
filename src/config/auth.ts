/**
 * Minimalist Auth Config
 */
export const authConfig = {
  accessTokenExpiry: 604800,
  
  cookies: {
    accessToken: 'zgi_access_token',
  },
  
  routes: {
    login: '/login',
    afterLogin: '/console',
    afterLogout: '/login',
  },
};

export const mockUsers = [
  { id: '1', email: 'admin@example.com', password: 'admin123', name: 'Admin', role: 'admin', avatar: null },
  { id: '2', email: 'user@example.com', password: 'user123', name: 'User', role: 'user', avatar: null },
];

export const generateMockToken = (userId: string, type: 'access' | 'refresh') => {
  return btoa(JSON.stringify({ userId, type, exp: Date.now() + 3600000 }));
};

export const decodeMockToken = (token: string) => {
  try {
    return JSON.parse(atob(token));
  } catch {
    return null;
  }
};
