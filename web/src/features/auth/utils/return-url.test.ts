import { describe, expect, it } from 'vitest';

import { authConfig } from '@/config/auth';
import { resolveReturnUrl } from './return-url';

describe('resolveReturnUrl', () => {
  it('keeps same-origin paths with query and hash', () => {
    expect(resolveReturnUrl('/console/settings?tab=api#keys')).toBe('/console/settings?tab=api#keys');
  });

  it('falls back for empty, absolute, and protocol-relative URLs', () => {
    expect(resolveReturnUrl(null)).toBe(authConfig.routes.afterLogin);
    expect(resolveReturnUrl('https://example.com/console')).toBe(authConfig.routes.afterLogin);
    expect(resolveReturnUrl('//example.com/console')).toBe(authConfig.routes.afterLogin);
  });

  it('falls back for slash-backslash browser confusion payloads', () => {
    expect(resolveReturnUrl('/\\example.com/console')).toBe(authConfig.routes.afterLogin);
  });
});
