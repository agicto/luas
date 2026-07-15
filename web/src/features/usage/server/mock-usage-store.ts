import 'server-only';

import type { AuthUser } from '@/features/auth/types';
import type {
  OrganizationUsageSummary,
  UsageMetric,
  UsageSummary,
  UserUsageSummary,
} from '@/features/usage/types';

const metricCatalog = [
  ['api.requests', 'request'],
  ['ai.input_tokens', 'token'],
  ['ai.output_tokens', 'token'],
  ['asset.transfer_bytes', 'byte'],
  ['workflow.runs', 'run'],
] as const satisfies readonly [UsageMetric, UsageSummary['unit']][];

export const mockUsageStore = {
  user(_user: AuthUser, at = new Date()): UserUsageSummary[] {
    return summaries('user', at, [286, 12_480, 3_106, 8_388_608, 14]) as UserUsageSummary[];
  },

  organization(_organizationId: number, at = new Date()): OrganizationUsageSummary[] {
    return summaries('organization', at, [1_842, 84_120, 19_604, 52_428_800, 61]) as OrganizationUsageSummary[];
  },
};

function summaries(
  scope: UsageSummary['scope'],
  at: Date,
  values: readonly number[]
): UsageSummary[] {
  const periodStart = new Date(Date.UTC(at.getUTCFullYear(), at.getUTCMonth(), 1));
  const periodEnd = new Date(Date.UTC(at.getUTCFullYear(), at.getUTCMonth() + 1, 1));
  return metricCatalog.map(([metric, unit], index) => {
    const used = values[index] ?? 0;
    const limit = index === 0 ? (scope === 'user' ? 1_000 : 5_000) : null;
    return {
      scope,
      metric,
      unit,
      period: 'month',
      period_start: periodStart.toISOString(),
      period_end: periodEnd.toISOString(),
      used,
      limit,
      remaining: limit === null ? null : Math.max(limit - used, 0),
      overage: limit === null ? 0 : Math.max(used - limit, 0),
      over_limit: limit !== null && used > limit,
      quota_source: limit === null ? 'default' : 'override',
      quota_version: limit === null ? 0 : 1,
      updated_at: used === 0 ? null : at.toISOString(),
    };
  });
}
