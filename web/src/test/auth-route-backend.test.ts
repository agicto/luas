import { describe, expect, it } from 'vitest';

import { resolveAuthRoute } from '@/app/api/_shared/auth-route';
import { ApiErrorCode } from '@/http/codes';

describe('auth route backend resolution', () => {
  it('prefers the production adapter when it is enabled', () => {
    expect(
      resolveAuthRoute({
        adapterEnabled: true,
        enabled: true,
        nodeEnv: 'production',
      })
    ).toEqual({ available: true, backend: 'go-api' });
  });

  it('uses the mock BFF only when mock behavior is available', () => {
    expect(
      resolveAuthRoute({
        adapterEnabled: false,
        enabled: false,
        nodeEnv: 'development',
      })
    ).toEqual({ available: true, backend: 'mock' });
  });

  it('returns a canonical 503 when production has no auth backend', async () => {
    const resolution = resolveAuthRoute({
      adapterEnabled: false,
      enabled: false,
      nodeEnv: 'production',
    });

    expect(resolution.available).toBe(false);
    if (resolution.available) {
      throw new Error('Expected auth route resolution to be unavailable');
    }

    expect(resolution.response.status).toBe(503);
    expect(resolution.response.headers.get('cache-control')).toBe(
      'private, no-store'
    );
    expect(resolution.response.headers.get('vary')).toContain('Cookie');
    await expect(resolution.response.json()).resolves.toEqual({
      code: 503,
      error_code: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message: 'Authentication backend is unavailable',
    });
  });
});
