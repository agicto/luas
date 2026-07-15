import request, { ApiError } from '@/http/request';
import { ClientErrorCode } from '@/http/codes';
import { usageSummaryListSchema } from '@/features/usage/schemas';
import type {
  OrganizationUsageSummary,
  UsageSummary,
  UserUsageSummary,
} from '@/features/usage/types';

const expectedMetrics = [
  'api.requests',
  'ai.input_tokens',
  'ai.output_tokens',
  'asset.transfer_bytes',
  'workflow.runs',
] as const;

export const usageService = {
  async user(): Promise<UserUsageSummary[]> {
    return parseUserUsage(await request.get<unknown>('/usage/user'));
  },

  async organization(organizationId: number): Promise<OrganizationUsageSummary[]> {
    if (!Number.isSafeInteger(organizationId) || organizationId < 1) throw invalidResponse();
    return parseOrganizationUsage(
      await request.get<unknown>('/organization-usage', {
        headers: { 'Organization-Id': String(organizationId) },
      })
    );
  },
};

export function parseUserUsage(value: unknown): UserUsageSummary[] {
  return parseUsage(value, 'user') as UserUsageSummary[];
}

export function parseOrganizationUsage(value: unknown): OrganizationUsageSummary[] {
  return parseUsage(value, 'organization') as OrganizationUsageSummary[];
}

function parseUsage(value: unknown, scope: UsageSummary['scope']): UsageSummary[] {
  const parsed = usageSummaryListSchema.safeParse(value);
  if (!parsed.success || parsed.data.some(item => item.scope !== scope)) throw invalidResponse();
  const metrics = parsed.data.map(item => item.metric);
  if (
    metrics.length !== expectedMetrics.length ||
    !expectedMetrics.every(metric => metrics.includes(metric))
  ) {
    throw invalidResponse();
  }
  return parsed.data;
}

function invalidResponse(): ApiError {
  return new ApiError('Usage service returned an invalid response', ClientErrorCode.INVALID_RESPONSE);
}
