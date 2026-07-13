export type AuthRole = 'admin' | 'member';

export interface AuthUser {
  id: string;
  email: string;
  name: string;
  role: AuthRole;
}

/**
 * Serializable result passed from a protected Server Component to its
 * request-scoped client auth store.
 */
export type AuthBootstrap =
  | { status: 'authenticated'; user: AuthUser }
  | { status: 'unauthenticated' }
  | { status: 'client-required' };

export type AuthStatus =
  | 'idle'
  | 'loading'
  | 'authenticated'
  | 'unauthenticated'
  | 'forbidden'
  | 'unavailable';

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  name: string;
  email: string;
  password: string;
}

export interface AuthResponse {
  user: AuthUser;
}

export interface LogoutResponse {
  success: true;
}
