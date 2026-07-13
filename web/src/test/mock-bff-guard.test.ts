import { describe, expect, it } from 'vitest';

import {
  guardMockBffRoute,
  guardSameOriginMutation,
  isMockBffEnabled,
} from '@/app/api/_shared/mock-bff';
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

describe('mock BFF same-origin mutation guard', () => {
  it('allows same-origin browser mutations', () => {
    const request = mutationRequest({
      origin: 'https://app.example.com',
      'sec-fetch-site': 'same-origin',
    });

    expect(guardSameOriginMutation(request)).toBeNull();
  });

  it('allows clients without browser fetch metadata', () => {
    const request = mutationRequest();

    expect(guardSameOriginMutation(request)).toBeNull();
  });

  it.each([
    {
      label: 'cross-site fetch metadata',
      headers: { 'sec-fetch-site': 'cross-site' },
    },
    {
      label: 'same-site sibling origin',
      headers: {
        origin: 'https://docs.example.com',
        'sec-fetch-site': 'same-site',
      },
    },
    {
      label: 'mismatched origin',
      headers: { origin: 'https://attacker.example' },
    },
    {
      label: 'opaque origin',
      headers: { origin: 'null' },
    },
  ])('rejects $label with the canonical forbidden response', async ({ headers }) => {
    const request = mutationRequest(headers);
    const response = guardSameOriginMutation(request);

    expect(response).not.toBeNull();
    if (!response) {
      throw new Error('Expected cross-origin mutation to be rejected');
    }

    expect(response.status).toBe(403);
    await expect(response.json()).resolves.toEqual({
      code: 403,
      error_code: ApiErrorCode.AUTH_FORBIDDEN,
      message: 'Cross-origin mutation is not allowed',
    });
  });
});

function mutationRequest(headers: Record<string, string | undefined> = {}): Request {
  const request = new Request('https://app.example.com/api/auth/login', {
    method: 'POST',
  });

  for (const [name, value] of Object.entries(headers)) {
    if (value !== undefined) {
      request.headers.set(name, value);
    }
  }

  return request;
}
