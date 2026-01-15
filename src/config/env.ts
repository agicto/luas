/**
 * Centralized environment variables configuration.
 * 
 * This module provides a type-safe, validated interface for accessing environment
 * variables. It differentiates between public (client-side) and private (server-side)
 * variables.
 * 
 * Usage:
 *  import { env, publicEnv } from '@/config/env';
 *  const apiUrl = publicEnv.NEXT_PUBLIC_API_URL;
 */

import { z } from 'zod';

// ============================================================================
// Schema Definitions
// ============================================================================

/**
 * Public environment variables (exposed to the client via NEXT_PUBLIC_ prefix)
 */
const publicEnvSchema = z.object({
  // Application
  NEXT_PUBLIC_APP_URL: z.string().url().default('http://localhost:3000'),

  // API
  NEXT_PUBLIC_API_URL: z.string().default('/api'),
  NEXT_PUBLIC_API_TIMEOUT: z.coerce.number().default(30000),

  // Authentication
  NEXT_PUBLIC_AUTH_TOKEN_MODE: z.enum(['basic', 'refresh']).default('refresh'),

  // i18n
  NEXT_PUBLIC_DEFAULT_LOCALE: z.enum(['zh-Hans', 'en-US']).default('zh-Hans'),
  NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED: z.preprocess(
    (val) => val !== 'false',
    z.boolean().default(true)
  ),

  // Analytics
  NEXT_PUBLIC_GA_MEASUREMENT_ID: z.string().optional(),

  // Optional: Upstream API (for BFF proxy or direct calls)
  NEXT_PUBLIC_UPSTREAM_API_BASE: z.string().url().optional(),
});

/**
 * Server-only environment variables
 */
const serverEnvSchema = z.object({
  NODE_ENV: z.enum(['development', 'production', 'test']).default('development'),

  // Internal API URL (used by SSR or API routes, not exposed to the client)
  INTERNAL_API_URL: z.string().url().optional(),

  // Auth config (server-side only)
  AUTH_ACCESS_TOKEN_EXPIRY: z.coerce.number().default(15 * 60), // 15 minutes
  AUTH_REFRESH_TOKEN_EXPIRY: z.coerce.number().default(7 * 24 * 60 * 60), // 7 days
});

// ============================================================================
// Parse and Validate
// ============================================================================

function getPublicEnv() {
  return publicEnvSchema.parse({
    NEXT_PUBLIC_APP_URL: process.env.NEXT_PUBLIC_APP_URL,
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL,
    NEXT_PUBLIC_API_TIMEOUT: process.env.NEXT_PUBLIC_API_TIMEOUT,
    NEXT_PUBLIC_AUTH_TOKEN_MODE: process.env.NEXT_PUBLIC_AUTH_TOKEN_MODE,
    NEXT_PUBLIC_DEFAULT_LOCALE: process.env.NEXT_PUBLIC_DEFAULT_LOCALE,
    NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED: process.env.NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED,
    NEXT_PUBLIC_GA_MEASUREMENT_ID: process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID,
    NEXT_PUBLIC_UPSTREAM_API_BASE: process.env.NEXT_PUBLIC_UPSTREAM_API_BASE,
  });
}

function getServerEnv() {
  // Only parse on the server side
  if (typeof window !== 'undefined') {
    return {} as z.infer<typeof serverEnvSchema>;
  }
  return serverEnvSchema.parse({
    NODE_ENV: process.env.NODE_ENV,
    INTERNAL_API_URL: process.env.INTERNAL_API_URL,
    AUTH_ACCESS_TOKEN_EXPIRY: process.env.AUTH_ACCESS_TOKEN_EXPIRY,
    AUTH_REFRESH_TOKEN_EXPIRY: process.env.AUTH_REFRESH_TOKEN_EXPIRY,
  });
}

// ============================================================================
// Exports
// ============================================================================

/**
 * Public environment variables, safe for client-side use.
 */
export const publicEnv = getPublicEnv();

/**
 * Server-only environment variables.
 * Accessing these on the client will return an empty object.
 */
export const serverEnv = getServerEnv();

/**
 * Convenience export combining both.
 * Prefer `publicEnv` for client code and `serverEnv` for server code.
 */
export const env = {
  ...publicEnv,
  ...serverEnv,
};

// Type exports for external use
export type PublicEnv = z.infer<typeof publicEnvSchema>;
export type ServerEnv = z.infer<typeof serverEnvSchema>;
