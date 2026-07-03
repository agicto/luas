import request from '@/http';
import type {
  TrendListResponse,
  TrendQuerySchema,
  TrendSummary,
  TrendSyncRun,
} from '@/features/trends/types';

/**
 * Trend Service
 *
 * Stateless API access for the AGI01 trend pipeline console.
 */
export const trendService = {
  getList: (params?: TrendQuerySchema) =>
    request.get<TrendListResponse>('/trends', { params }),

  getSummary: () =>
    request.get<TrendSummary>('/trend-stats'),

  createSyncRun: () =>
    request.post<TrendSyncRun>('/trend-sync-runs'),
};

export type TrendService = typeof trendService;
