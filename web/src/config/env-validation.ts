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

const integerEnv = z.preprocess((value) => {
  if (value === undefined || value === '') {
    return undefined;
  }

  if (typeof value === 'string' && /^\d+$/.test(value)) {
    return Number(value);
  }

  return value;
}, z.number().int());

const optionalString = <Schema extends z.ZodType>(schema: Schema) =>
  z.preprocess(
    (value) => (value === '' ? undefined : value),
    schema.optional()
  );

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
  AUTH_ADAPTER_ENABLED: booleanEnv.default(false),
  AUTH_API_TIMEOUT_MS: integerEnv.pipe(z.number().min(100).max(30_000)).default(5_000),
  AUTH_API_URL: optionalString(z.string().url()),
  AUTH_CLIENT_IP_HEADER: optionalString(
    z.string().regex(/^[a-z0-9!#$%&'*+.^_`|~-]+$/i).transform((value) => value.toLowerCase())
  ),
  MOCK_BFF_ENABLED: booleanEnv.default(false),
  SESSION_SECRET: optionalString(z.string().min(32)),
});
