import { NextResponse } from 'next/server';

interface ApiSuccessResponseOptions {
  status?: number;
  message?: string;
  headers?: HeadersInit;
}

export interface ApiPaginationMeta {
  current_page: number;
  per_page: number;
  total: number;
  last_page: number;
  from: number;
  to: number;
}

export interface ApiPaginationLinks {
  first: string;
  last: string;
  prev: string | null;
  next: string | null;
}

// apiSuccessResponse keeps mock BFF success envelopes aligned with the API contract.
export function apiSuccessResponse<T>(
  data: T,
  { status = 200, message = 'success', headers }: ApiSuccessResponseOptions = {}
): NextResponse {
  return NextResponse.json(
    {
      code: 0,
      message,
      data,
    },
    { status, headers }
  );
}

export function apiPaginatedResponse<T>(
  data: readonly T[],
  meta: ApiPaginationMeta,
  links: ApiPaginationLinks,
  headers?: HeadersInit
): NextResponse {
  return NextResponse.json({
    code: 0,
    message: 'success',
    data,
    meta,
    links,
  }, { headers });
}
