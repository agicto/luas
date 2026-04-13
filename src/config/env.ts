import { z } from 'zod';

/**
 * Extreme Purification: 
 * Only keep environment variables that MUST change between deploys.
 * Defaults are hardcoded as fallback.
 */
const envSchema = z.object({
  // Only the API entry point
  NEXT_PUBLIC_API_URL: z.string().min(1).default('/api'),
  NEXT_PUBLIC_DEPLOY_API_URL: z.string().min(1).default('http://localhost:8025/v1'),
  NEXT_PUBLIC_PLATFORM_API_URL: z.string().min(1).default('http://localhost:8025/v1'),

  // Absolute app URL for metadata, sitemap, and robots generation
  NEXT_PUBLIC_APP_URL: z.string().url().default('http://localhost:3000'),
  
  // Optional but sometimes required
  NEXT_PUBLIC_GA_MEASUREMENT_ID: z.string().optional(),
  
  // Server-only
  NODE_ENV: z.enum(['development', 'production', 'test']).default('development'),
});

const parsed = envSchema.safeParse({
  NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL,
  NEXT_PUBLIC_DEPLOY_API_URL: process.env.NEXT_PUBLIC_DEPLOY_API_URL,
  NEXT_PUBLIC_PLATFORM_API_URL:
    process.env.NEXT_PUBLIC_PLATFORM_API_URL ?? process.env.NEXT_PUBLIC_DEPLOY_API_URL,
  NEXT_PUBLIC_APP_URL: process.env.NEXT_PUBLIC_APP_URL,
  NEXT_PUBLIC_GA_MEASUREMENT_ID: process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID,
  NODE_ENV: process.env.NODE_ENV,
});

if (!parsed.success) {
  console.error('❌ Invalid environment variables:', parsed.error.format());
  throw new Error('Invalid environment variables');
}

export const env = parsed.data;

// Shorthands for cleaner calls
export const isDev = env.NODE_ENV === 'development';
export const isProd = env.NODE_ENV === 'production';
