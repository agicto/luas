import { authConfig } from '@/config/auth';

const SAME_ORIGIN_BASE = 'https://luas.local';

/**
 * @util resolveReturnUrl
 * @description Normalizes post-login return URLs to same-origin paths only.
 */
export function resolveReturnUrl(value: string | null): string {
  const raw = value?.trim();
  if (!raw || !raw.startsWith('/') || raw.startsWith('//') || raw.startsWith('/\\')) {
    return authConfig.routes.afterLogin;
  }

  try {
    const url = new URL(raw, SAME_ORIGIN_BASE);
    if (url.origin !== SAME_ORIGIN_BASE) {
      return authConfig.routes.afterLogin;
    }
    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return authConfig.routes.afterLogin;
  }
}
