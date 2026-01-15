// Authentication service layer
// Functional API pattern - stateless, pure functions

import { request } from '@/http';
import type {
  User,
  LoginRequest,
  RegisterRequest,
  SetupRequest,
  AuthResponse,
  SystemFeatures,
  SetupStatus,
} from '@/types/auth';

// API endpoints (Mock)
// All endpoints point to Next.js API routes in src/app/api/auth/*
const ENDPOINTS = {
  LOGIN: '/auth/login',
  LOGOUT: '/auth/logout',
  ME: '/auth/me',
  REGISTER: '/auth/register',
  SETUP_STATUS: '/auth/setup-status',
  SETUP: '/auth/setup',
  SYSTEM_FEATURES: '/auth/system-features',
} as const;

/**
 * Authentication API
 * All methods are stateless pure functions
 * 
 * Uses Next.js API routes in src/app/api/auth/* to simulate backend.
 */
export const authApi = {
  // System setup
  getSetupStatus: () =>
    request.get<SetupStatus>(ENDPOINTS.SETUP_STATUS),

  setup: (data: SetupRequest) =>
    request.post<AuthResponse>(ENDPOINTS.SETUP, data),

  getSystemFeatures: () =>
    request.get<SystemFeatures>(ENDPOINTS.SYSTEM_FEATURES),

  // Authentication
  login: async (credentials: LoginRequest & { remember?: boolean }) => {
    const result = await request.post<{ user: User }>(ENDPOINTS.LOGIN, credentials);
    return result.user;
  },

  register: (data: RegisterRequest) =>
    request.post<AuthResponse>(ENDPOINTS.REGISTER, data),

  logout: async () => {
    try {
      await request.post(ENDPOINTS.LOGOUT, {});
    } catch (error) {
      console.warn('Logout request failed:', error);
    }
  },

  // Profile
  getProfile: async () => {
    const result = await request.get<{ user: User }>(ENDPOINTS.ME);
    return result.user;
  },

  refreshToken: async (token: string) => {
    return request.post<{ accessToken: string; refreshToken: string }>(
      '/auth/refresh',
      { refreshToken: token },
      { skipAuth: true }
    );
  },
} as const;

// Type exports for external use
export type AuthApi = typeof authApi;