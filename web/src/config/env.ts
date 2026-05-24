import { z } from 'zod';

/**
 * Extreme Purification: 
 * Only keep environment variables that MUST change between deploys.
 * Defaults are hardcoded as fallback.
 */
const envSchema = z.object({
  // API entry point — point this at your Luas Go backend (or any backend)
  NEXT_PUBLIC_API_URL: z.string().min(1).default('/api'),

  // Absolute app URL for metadata, sitemap, and robots generation
  NEXT_PUBLIC_APP_URL: z.string().url().default('http://localhost:3000'),

  // Optional but sometimes required
  NEXT_PUBLIC_GA_MEASUREMENT_ID: z.string().optional(),

  // Server-only
  NODE_ENV: z.enum(['development', 'production', 'test']).default('development'),

  // Secret used to HMAC-sign the mock session cookie. Required in
  // production; a dev-only fallback is used otherwise (with a console
  // warning) so `pnpm dev` works out of the box.
  SESSION_SECRET: z.string().optional(),
});

const parsed = envSchema.safeParse({
  NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL,
  NEXT_PUBLIC_APP_URL: process.env.NEXT_PUBLIC_APP_URL,
  NEXT_PUBLIC_GA_MEASUREMENT_ID: process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID,
  NODE_ENV: process.env.NODE_ENV,
  SESSION_SECRET: process.env.SESSION_SECRET,
});

if (!parsed.success) {
  console.error('Invalid environment variables:', parsed.error.format());
  throw new Error('Invalid environment variables');
}

export const env = parsed.data;

// Shorthands for cleaner calls
export const isDev = env.NODE_ENV === 'development';
export const isProd = env.NODE_ENV === 'production';
