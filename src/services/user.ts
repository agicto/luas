// User service layer
// Example of functional API pattern with Zod validation for type safety

import { z } from 'zod';
import { request } from '@/http';
import { components } from '@/types/api.generated';

// ============================================================================
// Types - Refenced from generated schema for single source of truth
// ============================================================================

export type User = components['schemas']['User'];
export type UserListResponse = components['schemas']['UserListResponse'];

// ============================================================================
// Zod Schemas - Runtime validation matching the generated types
// ============================================================================

/**
 * User schema - validates API response structure
 * Use .parse() for strict validation (throws on invalid data)
 */
export const UserSchema = z.object({
  id: z.string(),
  name: z.string(),
  email: z.string(),
  avatar: z.string().optional(),
  role: z.enum(['admin', 'user']).optional(),
});

export const UserListResponseSchema = z.object({
  data: z.array(UserSchema),
  total: z.number(),
});

// Request DTOs (not from API, so no schema needed)
export interface CreateUserDto {
  name: string;
  email: string;
  password?: string;
}

export interface UpdateUserDto {
  name?: string;
  email?: string;
  avatar?: string;
}

export interface UserListParams {
  page?: number;
  limit?: number;
  search?: string;
}

// ============================================================================
// API Endpoints
// ============================================================================

const ENDPOINTS = {
  LIST: '/backend/api/users',
  DETAIL: (id: string) => `/backend/api/users/${id}`,
} as const;

// ============================================================================
// User API - Functional pattern with Zod validation
// ============================================================================

/**
 * User API
 * 
 * All responses are validated with Zod schemas to ensure type safety at runtime.
 * If the API returns malformed data, an error is thrown instead of silent failures.
 * 
 * @example
 * ```tsx
 * const user = await userApi.get('123');
 * // user is guaranteed to have id, name, email
 * ```
 */
export const userApi = {
  /**
   * List users with pagination
   */
  list: async (params?: UserListParams): Promise<UserListResponse> => {
    const response = await request.get(ENDPOINTS.LIST, { params });
    return UserListResponseSchema.parse(response);
  },

  /**
   * Get single user by ID
   */
  get: async (id: string): Promise<User> => {
    const response = await request.get(ENDPOINTS.DETAIL(id));
    return UserSchema.parse(response);
  },

  /**
   * Create a new user
   */
  create: async (data: CreateUserDto): Promise<User> => {
    const response = await request.post(ENDPOINTS.LIST, data);
    return UserSchema.parse(response);
  },

  /**
   * Update an existing user
   */
  update: async (id: string, data: UpdateUserDto): Promise<User> => {
    const response = await request.patch(ENDPOINTS.DETAIL(id), data);
    return UserSchema.parse(response);
  },

  /**
   * Delete a user
   */
  delete: async (id: string): Promise<void> => {
    await request.delete(ENDPOINTS.DETAIL(id));
  },
} as const;

export type UserApi = typeof userApi;
