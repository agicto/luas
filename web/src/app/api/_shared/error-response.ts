import { NextResponse } from 'next/server';
import type { ZodError } from 'zod';
import { ApiErrorCode, type ApiErrorCodeValue } from '@/http/codes';

interface ApiErrorResponseOptions {
  status: number;
  errorCode: ApiErrorCodeValue;
  message: string;
  errors?: Record<string, string[]>;
  headers?: HeadersInit;
  requestId?: string;
}

export function apiErrorResponse({
  status,
  errorCode,
  message,
  errors,
  headers,
  requestId,
}: ApiErrorResponseOptions): NextResponse {
  return NextResponse.json(
    {
      code: status,
      error_code: errorCode,
      message,
      ...(errors ? { errors } : {}),
      ...(requestId ? { request_id: requestId } : {}),
    },
    { status, headers }
  );
}

export function apiInvalidInputResponse(message: string): NextResponse {
  return apiErrorResponse({
    status: 400,
    errorCode: ApiErrorCode.COMMON_INVALID_INPUT,
    message,
  });
}

export function apiValidationErrorResponse(message: string, error?: ZodError): NextResponse {
  return apiErrorResponse({
    status: 422,
    errorCode: ApiErrorCode.COMMON_VALIDATION_FAILED,
    message,
    ...(error ? { errors: zodFieldErrors(error) } : {}),
  });
}

export function apiNotFoundResponse(message: string): NextResponse {
  return apiErrorResponse({
    status: 404,
    errorCode: ApiErrorCode.COMMON_NOT_FOUND,
    message,
  });
}

function zodFieldErrors(error: ZodError): Record<string, string[]> {
  return error.issues.reduce<Record<string, string[]>>((fieldErrors, issue) => {
    const field = issue.path.join('.') || 'body';
    fieldErrors[field] = [...(fieldErrors[field] ?? []), issue.message];
    return fieldErrors;
  }, {});
}
