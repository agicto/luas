import request from '@/http';
import type { AuthResponse, LoginRequest, RegisterRequest } from '@/features/auth/types';

export const authService = {
  login: (data: LoginRequest) =>
    request.post<AuthResponse, LoginRequest>('/auth/login', data, {
      skipErrorHandler: true,
    }),

  register: (data: RegisterRequest) =>
    request.post<AuthResponse, RegisterRequest>('/auth/register', data, {
      skipErrorHandler: true,
    }),

  me: () =>
    request.get<AuthResponse>('/auth/me', {
      skipErrorHandler: true,
    }),

  logout: () =>
    request.post<{ success: true }, undefined>('/auth/logout', undefined, {
      skipErrorHandler: true,
    }),
};

export type AuthService = typeof authService;
