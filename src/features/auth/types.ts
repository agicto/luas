export type AuthRole = 'admin' | 'member';

export interface AuthUser {
  id: string;
  email: string;
  name: string;
  role: AuthRole;
}

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
