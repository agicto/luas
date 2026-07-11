import 'server-only';

import { z } from 'zod';

import { locales } from '@/i18n/locales';

const booleanEnv = z.preprocess((value) => {
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

export const publicEnvSchema = z.object({
  NEXT_PUBLIC_API_URL: z.string().min(1),
  NEXT_PUBLIC_APP_URL: z.string().url(),
  NEXT_PUBLIC_GA_MEASUREMENT_ID: z.string().optional(),
  NEXT_PUBLIC_DEFAULT_LOCALE: z.enum(locales),
  NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED: z.boolean(),
  NODE_ENV: z.enum(['development', 'production', 'test']),
});

export type PublicEnv = z.infer<typeof publicEnvSchema>;

export const serverEnvSchema = z.object({
  MOCK_BFF_ENABLED: booleanEnv.default(false),
  SESSION_SECRET: z.preprocess(
    (value) => (value === '' ? undefined : value),
    z.string().min(32).optional()
  ),
});
