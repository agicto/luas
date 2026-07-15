import { describe, expect, it } from 'vitest';

import {
  privateNoStoreHeaders,
  privateNoStoreResponse,
} from '@/server/http/private-response';

describe('private no-store response boundary', () => {
  it('preserves existing headers and merges Vary case-insensitively', () => {
    const headers = privateNoStoreHeaders(
      {
        'cache-control': 'public, max-age=300',
        'retry-after': '30',
        vary: 'Origin, Organization-Id',
      },
      ['Cookie', 'organization-id', 'COOKIE']
    );

    expect(headers.get('cache-control')).toBe('private, no-store');
    expect(headers.get('retry-after')).toBe('30');
    expect(headers.get('vary')).toBe('Origin, Organization-Id, Cookie');
  });

  it('applies the policy to an existing response without replacing Vary', () => {
    const response = privateNoStoreResponse(
      Response.json(
        { ok: true },
        { headers: { vary: 'Accept-Language', 'x-request-id': 'req-1' } }
      ),
      ['Cookie']
    );

    expect(response.headers.get('cache-control')).toBe('private, no-store');
    expect(response.headers.get('vary')).toBe('Accept-Language, Cookie');
    expect(response.headers.get('x-request-id')).toBe('req-1');
  });

  it('leaves a wildcard Vary policy intact', () => {
    const headers = privateNoStoreHeaders({ vary: '*' }, ['Cookie']);

    expect(headers.get('vary')).toBe('*');
  });
});
