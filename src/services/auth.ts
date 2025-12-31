// Authentication service layer
// Functional API pattern - stateless, pure functions

import { request } from '@/http';
import { authConfig } from '@/config/auth';
import type {
  User,
  LoginRequest,
  RegisterRequest,
  SetupRequest,
  AuthResponse,
  SystemFeatures,
  SetupStatus,
} from '@/types/auth';

// API endpoints (BFF layer)
// Switches between production and mock endpoints based on config
const getEndpoints = () => {
  const prefix = authConfig.useMockAuth ? '/auth/mock' : '/auth';
  return {
    LOGIN: `${prefix}/login`,
    LOGOUT: `${prefix}/logout`,
    ME: `${prefix}/me`,
    // These are only available in production mode
    REGISTER: '/auth/register',
    SETUP_STATUS: '/auth/setup-status',
    SETUP: '/auth/setup',
    SYSTEM_FEATURES: '/auth/system-features',
    PROFILE: '/backend/api/user/profile',
  };
};

const ENDPOINTS = getEndpoints();

/**
 * Authentication API
 * All methods are stateless pure functions
 * 
 * When NEXT_PUBLIC_USE_MOCK_AUTH=true, uses mock endpoints.
 * Otherwise uses production endpoints that proxy to upstream.
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

  updateProfile: (data: Partial<User>) =>
    request.patch<User>(ENDPOINTS.PROFILE, data),
} as const;

// Type exports for external use
export type AuthApi = typeof authApi;