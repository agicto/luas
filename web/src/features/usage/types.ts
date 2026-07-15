import type { infer as Infer } from 'zod/mini';

import type { usageMetricSchema, usageSummarySchema } from './schemas';

export type UsageMetric = Infer<typeof usageMetricSchema>;
export type UsageSummary = Infer<typeof usageSummarySchema>;
export type UserUsageSummary = UsageSummary & { scope: 'user' };
export type OrganizationUsageSummary = UsageSummary & { scope: 'organization' };
