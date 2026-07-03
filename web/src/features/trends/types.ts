export type TrendStatus = 'new' | 'candidate' | 'selected' | 'rejected' | 'archived';

export interface TrendSource {
  provider: string;
  name: string;
}

export interface TrendScores {
  h_score?: number;
  k_score?: number;
  r_score?: number;
  brand_fit_score?: number;
  risk_score?: number;
  total_score?: number;
  recommended?: boolean;
}

export interface TrendItem {
  id: string;
  source: TrendSource;
  canonical_url: string;
  title: string;
  summary?: string;
  channel?: string;
  language: string;
  highlighted_at?: string;
  significance?: string;
  status: TrendStatus;
  scores: TrendScores;
  target_audience?: string;
  recommended_angle?: string;
  risk_notes?: string;
  evaluated_at?: string;
  created_at: string;
  updated_at: string;
}

export interface PaginationMeta {
  current_page: number;
  per_page: number;
  total: number;
  last_page: number;
  from: number;
  to: number;
}

export interface PaginationLinks {
  first: string;
  last: string;
  prev?: string | null;
  next?: string | null;
}

export interface TrendListResponse {
  items: TrendItem[];
  meta: PaginationMeta | null;
  links: PaginationLinks | null;
}

export interface TrendStats {
  total: number;
  new: number;
  candidate: number;
  selected: number;
  rejected: number;
  queued_jobs: number;
  succeeded_jobs: number;
  latest_trend_at?: string;
  latest_polled_at?: string;
  latest_evaluated_at?: string;
}

export interface TrendSourceStatus {
  id?: string;
  provider?: string;
  name?: string;
  source_url?: string;
  poll_interval_minutes?: number;
  status?: string;
  last_polled_at?: string;
  last_error?: string;
}

export interface TrendSummary {
  stats: TrendStats;
  source?: TrendSourceStatus;
}

export interface TrendQuerySchema {
  status?: TrendStatus;
  channel?: string;
  search?: string;
  recommended?: boolean;
  page?: number;
  per_page?: number;
}

export interface TrendSyncRun {
  source_id: string;
  fetched: number;
  upserted: number;
  inserted: number;
  evaluated: number;
  candidates: number;
  enqueued_score_job: number;
}
