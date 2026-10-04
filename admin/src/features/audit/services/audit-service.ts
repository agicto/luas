import type { z } from 'zod';
import { assertContract, type Accepts, type ContractResponse } from '@/http/contract';
import { http } from '@/http/client';
import { auditPageSchema, type AuditPage, type AuditSearch } from '@/features/audit/types';

/** Converts a local calendar day to the instant it starts, shifted by `days`. */
function startOfLocalDay(day: string, days = 0): string {
  const [year, month, date] = day.split('-').map(Number);
  return new Date(year ?? 1970, (month ?? 1) - 1, (date ?? 1) + days).toISOString();
}

export const auditService = {
  list(search: AuditSearch, signal?: AbortSignal): Promise<AuditPage> {
    const params = new URLSearchParams({ page: String(search.page ?? 1), per_page: '50' });
    if (search.action) {
      params.set('action', search.action);
    }
    if (search.user_id) {
      params.set('user_id', String(search.user_id));
    }
    if (search.from) {
      params.set('from', startOfLocalDay(search.from));
    }
    if (search.to) {
      // The API end is exclusive, so an inclusive local day ends at the next day's start.
      params.set('to', startOfLocalDay(search.to, 1));
    }
    return http.get(`/v1/operator/audit-logs?${params.toString()}`, {
      responseMode: 'json',
      schema: auditPageSchema,
      ...(signal ? { signal } : {}),
    });
  },
};

// Fails type-check when contracts/openapi.yaml allows a body these schemas would reject.
assertContract<Accepts<z.input<typeof auditPageSchema>, ContractResponse<'listOperatorAuditLogs', 200>>>();
