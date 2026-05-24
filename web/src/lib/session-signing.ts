/**
 * HMAC-SHA256-signed session payloads.
 *
 * Used by the MOCK auth seam. The format is `<base64url(payload)>.<base64url(sig)>`
 * where `sig = HMAC-SHA256(payload, secret)`. Compatible with both the
 * Node runtime (server actions, route handlers, RSC) and the Edge runtime
 * (Next.js middleware) — both expose `globalThis.crypto.subtle`.
 *
 * IMPORTANT: This is still a MOCK session. It is signed (so the client
 * can't forge a payload), but it's not a real backend session — the
 * server has no record of having issued it, so it can't be revoked, and
 * it carries the full user record rather than just an opaque ID. When
 * you replace the mock backend with a real one, drop these helpers in
 * favor of an opaque session token issued by your API.
 */

import { env, isProd } from '@/config/env';

const FALLBACK_DEV_SECRET = 'luas-dev-only-session-secret-do-not-use-in-production';

let warnedAboutFallback = false;

function getSecret(): string {
  if (env.SESSION_SECRET && env.SESSION_SECRET.length > 0) {
    return env.SESSION_SECRET;
  }
  if (isProd) {
    throw new Error('SESSION_SECRET must be set in production');
  }
  if (!warnedAboutFallback) {
    console.warn(
      '[luas] SESSION_SECRET is not set — using a non-production fallback. ' +
        'Generate one with `openssl rand -hex 32` and set it in .env.local.',
    );
    warnedAboutFallback = true;
  }
  return FALLBACK_DEV_SECRET;
}

async function hmacKey(secret: string): Promise<CryptoKey> {
  return globalThis.crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign', 'verify'],
  );
}

function base64UrlEncode(bytes: ArrayBuffer): string {
  const arr = new Uint8Array(bytes);
  let bin = '';
  for (let i = 0; i < arr.length; i++) bin += String.fromCharCode(arr[i]);
  // btoa is available in both Node 18+ and Edge.
  return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function base64UrlDecode(s: string): ArrayBuffer {
  const pad = s.length % 4 === 0 ? '' : '='.repeat(4 - (s.length % 4));
  const b64 = s.replace(/-/g, '+').replace(/_/g, '/') + pad;
  const bin = atob(b64);
  const buf = new ArrayBuffer(bin.length);
  const view = new Uint8Array(buf);
  for (let i = 0; i < bin.length; i++) view[i] = bin.charCodeAt(i);
  return buf;
}

/**
 * Sign a payload with HMAC-SHA256 and return `<payload>.<sig>`.
 * Both halves are base64url-encoded.
 */
export async function signSession(payload: string): Promise<string> {
  const key = await hmacKey(getSecret());
  const payloadBytes = new TextEncoder().encode(payload);
  // .buffer.slice() guarantees a fresh ArrayBuffer (not SharedArrayBuffer).
  const payloadBuf = payloadBytes.buffer.slice(
    payloadBytes.byteOffset,
    payloadBytes.byteOffset + payloadBytes.byteLength,
  ) as ArrayBuffer;
  const sig = (await globalThis.crypto.subtle.sign('HMAC', key, payloadBuf)) as ArrayBuffer;
  return `${base64UrlEncode(payloadBuf)}.${base64UrlEncode(sig)}`;
}

/**
 * Verify a signed payload. Returns the original payload string if the
 * signature is valid, otherwise `null`. Uses constant-time comparison
 * via `crypto.subtle.verify`.
 */
export async function verifySession(signed: string | undefined | null): Promise<string | null> {
  if (!signed) return null;
  const dot = signed.lastIndexOf('.');
  if (dot <= 0 || dot === signed.length - 1) return null;

  const payloadB64 = signed.slice(0, dot);
  const sigB64 = signed.slice(dot + 1);

  let payloadBuf: ArrayBuffer;
  let sigBuf: ArrayBuffer;
  try {
    payloadBuf = base64UrlDecode(payloadB64);
    sigBuf = base64UrlDecode(sigB64);
  } catch {
    return null;
  }

  let ok = false;
  try {
    const key = await hmacKey(getSecret());
    ok = await globalThis.crypto.subtle.verify('HMAC', key, sigBuf, payloadBuf);
  } catch {
    return null;
  }

  if (!ok) return null;
  return new TextDecoder().decode(payloadBuf);
}
