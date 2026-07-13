import request from '@/http';
import type {
  AuthResponse,
  LoginRequest,
  LogoutResponse,
  RegisterRequest,
} from '@/features/auth/types';
import { isAuthResponse, isLogoutResponse } from '@/features/auth/utils/auth-response';
import { ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

async function validateResponse<T>(
  requestPromise: Promise<unknown>,
  isValid: (value: unknown) => value is T,
  contractName: string
): Promise<T> {
  const response = await requestPromise;

  if (!isValid(response)) {
    throw new ApiError(`Invalid ${contractName} response`, ClientErrorCode.INVALID_RESPONSE);
  }

  return response;
}

export const authService = {
  login: (data: LoginRequest) =>
    validateResponse<AuthResponse>(
      request.post<unknown, LoginRequest>('/auth/login', data),
      isAuthResponse,
      'login'
    ),

  register: (data: RegisterRequest) =>
    validateResponse<AuthResponse>(
      request.post<unknown, RegisterRequest>('/auth/register', data),
      isAuthResponse,
      'registration'
    ),

  me: () =>
    validateResponse<AuthResponse>(
      request.get<unknown>('/auth/me'),
      isAuthResponse,
      'current-session'
    ),

  logout: () =>
    validateResponse<LogoutResponse>(
      request.post<unknown, undefined>('/auth/logout'),
      isLogoutResponse,
      'logout'
    ),
};

export type AuthService = typeof authService;
