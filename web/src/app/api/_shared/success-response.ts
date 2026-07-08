import { NextResponse } from 'next/server';

interface ApiSuccessResponseOptions {
  status?: number;
  message?: string;
}

// apiSuccessResponse keeps mock BFF success envelopes aligned with the API contract.
export function apiSuccessResponse<T>(
  data: T,
  { status = 200, message = 'success' }: ApiSuccessResponseOptions = {}
): NextResponse {
  return NextResponse.json(
    {
      code: 0,
      message,
      data,
    },
    { status }
  );
}
