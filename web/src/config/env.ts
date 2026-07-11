import {
  defaultLocaleFallback,
  locales,
  type Locale,
} from '@/i18n/locales';
import type { PublicEnv } from './env-validation';

/**
 * Resolve browser-safe values without shipping the server-side Zod validator.
 * RootLayout imports server-env.ts so the same values are schema-validated
 * during builds and server startup.
 */
function invalidPublicEnv(name: string): never {
  throw new Error(`Invalid public environment variable: ${name}`);
}

function readRequiredString(
  name: string,
  value: string | undefined,
  fallback: string
): string {
  const resolved = value ?? fallback;
  return resolved.length > 0 ? resolved : invalidPublicEnv(name);
}

function readAbsoluteUrl(
  name: string,
  value: string | undefined,
  fallback: string
): string {
  const resolved = readRequiredString(name, value, fallback);
  try {
    new URL(resolved);
    return resolved;
  } catch {
    return invalidPublicEnv(name);
  }
}

function readBoolean(
  name: string,
  value: string | undefined,
  fallback: boolean
): boolean {
  if (value === undefined || value === '') {
    return fallback;
  }

  const normalized = value.toLowerCase();
  if (['true', '1', 'yes', 'on'].includes(normalized)) {
    return true;
  }
  if (['false', '0', 'no', 'off'].includes(normalized)) {
    return false;
  }
  return invalidPublicEnv(name);
}

function readLocale(value: string | undefined): Locale {
  const resolved = value ?? defaultLocaleFallback;
  return locales.includes(resolved as Locale)
    ? (resolved as Locale)
    : invalidPublicEnv('NEXT_PUBLIC_DEFAULT_LOCALE');
}

function readNodeEnv(value: string | undefined): PublicEnv['NODE_ENV'] {
  const resolved = value ?? 'development';
  return resolved === 'development' || resolved === 'production' || resolved === 'test'
    ? resolved
    : invalidPublicEnv('NODE_ENV');
}

export const env: PublicEnv = {
  NEXT_PUBLIC_API_URL: readRequiredString(
    'NEXT_PUBLIC_API_URL',
    process.env.NEXT_PUBLIC_API_URL,
    '/api'
  ),
  NEXT_PUBLIC_APP_URL: readAbsoluteUrl(
    'NEXT_PUBLIC_APP_URL',
    process.env.NEXT_PUBLIC_APP_URL,
    'http://localhost:3000'
  ),
  NEXT_PUBLIC_GA_MEASUREMENT_ID:
    process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID || undefined,
  NEXT_PUBLIC_DEFAULT_LOCALE: readLocale(
    process.env.NEXT_PUBLIC_DEFAULT_LOCALE
  ),
  NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED: readBoolean(
    'NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED',
    process.env.NEXT_PUBLIC_LOCALE_SWITCHER_ENABLED,
    true
  ),
  NODE_ENV: readNodeEnv(process.env.NODE_ENV),
};

export const isDev = env.NODE_ENV === 'development';
export const isProd = env.NODE_ENV === 'production';
