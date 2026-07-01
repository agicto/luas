import { describe, expect, it } from 'vitest';

import { guardMockBffRoute, isMockBffEnabled } from '@/app/api/_shared/mock-bff';
import { ApiErrorCode } from '@/http/codes';

describe('mock BFF production guard', () => {
  it('allows mock routes outside production by default', () => {
    expect(isMockBffEnabled({ nodeEnv: 'development', enabled: false })).toBe(true);
    expect(isMockBffEnabled({ nodeEnv: 'test', enabled: false })).toBe(true);
  });

  it('disables mock routes in production unless explicitly enabled', () => {
    expect(isMockBffEnabled({ nodeEnv: 'production', enabled: false })).toBe(false);
    expect(isMockBffEnabled({ nodeEnv: 'production', enabled: true })).toBe(true);
  });

  it('returns a contract-shaped 503 when production mock routes are disabled', async () => {
    const response = guardMockBffRoute({ nodeEnv: 'production', enabled: false });

    expect(response).not.toBeNull();
    if (!response) {
      throw new Error('Expected disabled production mock BFF route to return a response');
    }

    expect(response.status).toBe(503);
    await expect(response.json()).resolves.toEqual({
      code: 503,
      error_code: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message: 'Mock BFF is disabled in production runtime',
    });
  });
});
