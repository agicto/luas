import '@tanstack/react-query';

import type { QueryOperationMeta } from '@/config/query-meta';

declare module '@tanstack/react-query' {
  interface Register {
    mutationMeta: QueryOperationMeta;
    queryMeta: QueryOperationMeta;
  }
}
