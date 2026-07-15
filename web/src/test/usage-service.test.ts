import { describe, expect, it } from 'vitest';

import {
  parseOrganizationUsage,
  parseUserUsage,
} from '@/features/usage/services/usage-service';

describe('usage service contract', () => {
  it('accepts the exact finite user and organization catalogs', () => {
    expect(parseUserUsage(usageList('user'))).toHaveLength(5);
    expect(parseOrganizationUsage(usageList('organization'))).toHaveLength(5);
  });

  it('rejects missing, duplicate, mixed-scope, and unknown metrics', () => {
    const values = usageList('user');
    expect(() => parseUserUsage(values.slice(1))).toThrow();
    expect(() => parseUserUsage([...values.slice(0, 4), values[0]])).toThrow();
    expect(() =>
      parseUserUsage([{ ...values[0], scope: 'organization' }, ...values.slice(1)])
    ).toThrow();
    expect(() =>
      parseUserUsage([{ ...values[0], metric: 'custom.events' }, ...values.slice(1)])
    ).toThrow();
  });

  it('rejects unsafe integers and inconsistent quota derivations', () => {
    const values = usageList('user');
    expect(() =>
      parseUserUsage([{ ...values[0], used: Number.MAX_SAFE_INTEGER + 1 }, ...values.slice(1)])
    ).toThrow();
    expect(() =>
      parseUserUsage([{ ...values[0], remaining: 999 }, ...values.slice(1)])
    ).toThrow();
    expect(() =>
      parseUserUsage([{ ...values[0], over_limit: true }, ...values.slice(1)])
    ).toThrow();
    expect(() =>
      parseUserUsage([
        { ...values[1], quota_source: 'override', limit: null },
        values[0],
        ...values.slice(2),
      ])
    ).toThrow();
  });
});

function usageList(scope: 'user' | 'organization') {
  return [
    usage(scope, 'api.requests', 'request', 286, 1_000),
    usage(scope, 'ai.input_tokens', 'token', 12_480, null),
    usage(scope, 'ai.output_tokens', 'token', 3_106, null),
    usage(scope, 'asset.transfer_bytes', 'byte', 8_388_608, null),
    usage(scope, 'workflow.runs', 'run', 14, null),
  ];
}

function usage(
  scope: 'user' | 'organization',
  metric: string,
  unit: string,
  used: number,
  limit: number | null
) {
  return {
    scope,
    metric,
    unit,
    period: 'month',
    period_start: '2026-07-01T00:00:00Z',
    period_end: '2026-08-01T00:00:00Z',
    used,
    limit,
    remaining: limit === null ? null : Math.max(limit - used, 0),
    overage: limit === null ? 0 : Math.max(used - limit, 0),
    over_limit: limit !== null && used > limit,
    quota_source: limit === null ? 'default' : 'override',
    quota_version: limit === null ? 0 : 1,
    updated_at: '2026-07-15T12:00:00Z',
  };
}
