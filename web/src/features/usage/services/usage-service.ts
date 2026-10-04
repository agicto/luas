import type { z } from 'zod/mini';
import { assertContract, type Accepts, type ContractData } from '@/http/contract';
import request, { ApiError } from '@/http/request';
import { ClientErrorCode } from '@/http/codes';
import { usageSummarySchema, usageSummaryWireListSchema } from '@/features/usage/schemas';
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

/**
 * Returns the rendered metrics in catalog order. Each must be present and valid; metrics the server
 * adds beyond them are ignored so an API with a larger catalog does not break this page.
 */
function parseUsage(value: unknown, scope: UsageSummary['scope']): UsageSummary[] {
  const wire = usageSummaryWireListSchema.safeParse(value);
  if (!wire.success || wire.data.some(item => item.scope !== scope)) throw invalidResponse();
  return expectedMetrics.map(metric => {
    const parsed = usageSummarySchema.safeParse(wire.data.find(item => item.metric === metric));
    if (!parsed.success) throw invalidResponse();
    return parsed.data;
  });
}

function invalidResponse(): ApiError {
  return new ApiError(
    'Usage service returned an invalid response',
    ClientErrorCode.INVALID_RESPONSE
  );
}

assertContract<
  Accepts<z.input<typeof usageSummaryWireListSchema>, ContractData<'listUserUsage', 200>>
>();
assertContract<
  Accepts<z.input<typeof usageSummaryWireListSchema>, ContractData<'listOrganizationUsage', 200>>
>();
