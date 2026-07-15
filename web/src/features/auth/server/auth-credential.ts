import 'server-only';

import { AUTH_SESSION_MAX_AGE_SECONDS } from '@/config/auth-session';

const OPAQUE_AUTHENTICATION_CREDENTIAL_PATTERN = /^[A-Za-z0-9_-]{43}$/;

export function isOpaqueAuthenticationCredential(value: unknown): value is string {
  if (
    typeof value !== 'string' ||
    !OPAQUE_AUTHENTICATION_CREDENTIAL_PATTERN.test(value)
  ) {
    return false;
  }

  return Buffer.from(value, 'base64url').byteLength === 32;
}

export function authenticationSessionMaxAgeSeconds(value: unknown): number | null {
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value <= 0) {
    return null;
  }

  return Math.min(value, AUTH_SESSION_MAX_AGE_SECONDS);
}
