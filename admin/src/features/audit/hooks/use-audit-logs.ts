import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { auditService } from '@/features/audit/services/audit-service';
import type { AuditSearch } from '@/features/audit/types';

export function useAuditLogs(search: AuditSearch) {
  return useQuery({
    queryKey: ['operator', 'audit-logs', search],
    queryFn: ({ signal }) => auditService.list(search, signal),
    placeholderData: keepPreviousData,
  });
}
