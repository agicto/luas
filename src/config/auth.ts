// Authentication configuration
// Configures token modes, expiry times, and mock users for development

// ============================================================================
// Token Mode Configuration
// ============================================================================

export type TokenMode = 'basic' | 'refresh';

/**
 * Auth configuration
 * 
 * Token Modes:
 * - 'basic': Single access token, no refresh (simpler, shorter sessions)
 * - 'refresh': Access + refresh token pair (production, longer sessions)
 */
export const authConfig = {
  /**
   * Token mode: 'basic' | 'refresh'
   * Set via NEXT_PUBLIC_AUTH_TOKEN_MODE env variable
   */
  tokenMode: (process.env.NEXT_PUBLIC_AUTH_TOKEN_MODE || 'refresh') as TokenMode,

  /**
   * Access token expiry in seconds
   * Default: 15 minutes
   */
  accessTokenExpiry: Number(process.env.AUTH_ACCESS_TOKEN_EXPIRY || 15 * 60),

  /**
   * Refresh token expiry in seconds
   * Default: 7 days
   */
  refreshTokenExpiry: Number(process.env.AUTH_REFRESH_TOKEN_EXPIRY || 7 * 24 * 60 * 60),

  /**
   * Cookie names for tokens
   */
  cookies: {
    accessToken: 'scaffold_access_token',
    refreshToken: 'scaffold_refresh_token',
  },

  /**
   * Routes configuration
   */
  routes: {
    login: '/login',
    afterLogin: '/console',
    afterLogout: '/login',
  },
} as const;

// ============================================================================
// Mock Users for Development
// ============================================================================

export interface MockUser {
  id: string;
  email: string;
  password: string;
  name: string;
  role: 'admin' | 'user';
  avatar?: string;
}

/**
 * Mock users for development/testing
 * Only used by mock auth endpoints
 */
export const mockUsers: MockUser[] = [
  {
    id: 'mock-admin-001',
    email: 'admin@example.com',
    password: 'admin123',
    name: 'Admin User',
    role: 'admin',
  },
  {
    id: 'mock-user-001',
    email: 'user@example.com',
    password: 'user123',
    name: 'Test User',
    role: 'user',
  },
];

// ============================================================================
// Token Generation Utilities (for mock API)
// ============================================================================

/**
 * Generate a simple mock token
 * In production, use proper JWT library
 */
export function generateMockToken(userId: string, type: 'access' | 'refresh'): string {
  const payload = {
    userId,
    type,
    iat: Date.now(),
    exp: Date.now() + (type === 'access' ? authConfig.accessTokenExpiry : authConfig.refreshTokenExpiry) * 1000,
  };
  // Simple base64 encoding for mock purposes
  // In production, use proper JWT with secret
  return Buffer.from(JSON.stringify(payload)).toString('base64');
}

/**
 * Decode and validate a mock token
 */
export function decodeMockToken(token: string): { userId: string; type: string; exp: number } | null {
  try {
    const decoded = JSON.parse(Buffer.from(token, 'base64').toString('utf-8'));
    if (decoded.exp < Date.now()) {
      return null; // Token expired
    }
    return decoded;
  } catch {
    return null;
  }
}

/**
 * Check if refresh mode is enabled
 */
export function isRefreshModeEnabled(): boolean {
  return authConfig.tokenMode === 'refresh';
}
