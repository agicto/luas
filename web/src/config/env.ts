import { z } from 'zod';

import { defaultLocaleFallback, locales } from '@/i18n/locales';

export const booleanEnv = z.preprocess((value) => {
  if (value === undefined || value === '') {
    return undefined;
  }

  if (typeof value === 'string') {
    const normalized = value.toLowerCase();

    if (['true', '1', 'yes', 'on'].includes(normalized)) {
      return true;
    }

    if (['false', '0', 'no', 'off'].includes(normalized)) {
      return false;
    }
  }

  return value;
}, z.boolean());

/**
 * Keep environment variables limited to values that may change between deploys.
 * Runtime code should import this module instead of reading process.env directly.
 */
const envSchema = z.object({
  // API entry point — point this at your Luas Go backend (or any backend)
  NEXT_PUBLIC_API_URL: z.string().min(1).default('/api'),

  // Absolute app URL for metadata, sitemap, and robots generation
  NEXT_PUBLIC_APP_URL: z.string().url().default('http://localhost:3000'),

  // Optional but sometimes required
  NEXT_PUBLIC_GA_MEASUREMENT_ID: z.string().optional(),

  // i18n
  NEXT_PUBLIC_DEFAULT_LOCALE: z.enum(locales).default(defaultLocaleFallback),
  NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED: booleanEnv.default(true),

  // Server-only
  NODE_ENV: z.enum(['development', 'production', 'test']).default('development'),

  // Keep this in the client-safe env because React devtools and logging
  // toggles need it in browser bundles. Server-only values live in
  // `env.server.ts`.
});

const parsed = envSchema.safeParse({
  NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL,
  NEXT_PUBLIC_APP_URL: process.env.NEXT_PUBLIC_APP_URL,
  NEXT_PUBLIC_GA_MEASUREMENT_ID: process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID,
  NEXT_PUBLIC_DEFAULT_LOCALE: process.env.NEXT_PUBLIC_DEFAULT_LOCALE,
  NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED: process.env.NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED,
  NODE_ENV: process.env.NODE_ENV,
});

if (!parsed.success) {
  console.error('Invalid environment variables:', parsed.error.format());
  throw new Error('Invalid environment variables');
}

export const env = parsed.data;

// Shorthands for cleaner calls
export const isDev = env.NODE_ENV === 'development';
export const isProd = env.NODE_ENV === 'production';
