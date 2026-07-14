import 'server-only';

const MAX_COMPACT_JWT_LENGTH = 3_500;
const COMPACT_JWT_PATTERN = /^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/;

export function isCompactJwt(value: unknown): value is string {
  return (
    typeof value === 'string' &&
    value.length > 0 &&
    value.length <= MAX_COMPACT_JWT_LENGTH &&
    COMPACT_JWT_PATTERN.test(value)
  );
}

export function authTokenMaxAgeSeconds(
  token: string,
  nowMilliseconds = Date.now()
): number | null {
  if (!isCompactJwt(token)) {
    return null;
  }

  try {
    const encodedPayload = token.split('.')[1];
    const payload = JSON.parse(
      Buffer.from(encodedPayload, 'base64url').toString('utf8')
    ) as unknown;

    if (typeof payload !== 'object' || payload === null) {
      return null;
    }

    const exp = (payload as Record<string, unknown>).exp;
    if (typeof exp !== 'number' || !Number.isSafeInteger(exp)) {
      return null;
    }

    const remaining = exp - Math.floor(nowMilliseconds / 1_000);
    return remaining > 0 ? remaining : null;
  } catch {
    return null;
  }
}
