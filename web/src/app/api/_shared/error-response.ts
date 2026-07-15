import { NextResponse } from 'next/server';
import { ApiErrorCode, type ApiErrorCodeValue } from '@/http/codes';

interface ValidationError {
  issues: readonly {
    path: readonly PropertyKey[];
    message: string;
  }[];
}

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

export function apiJsonBodyErrorResponse(
  error: 'invalid' | 'too_large'
): NextResponse {
  if (error === 'too_large') {
    return apiErrorResponse({
      status: 413,
      errorCode: ApiErrorCode.COMMON_REQUEST_TOO_LARGE,
      message: 'JSON body exceeds the allowed size',
    });
  }

  return apiInvalidInputResponse('Malformed JSON body');
}

export function apiValidationErrorResponse(
  message: string,
  error?: ValidationError
): NextResponse {
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

function zodFieldErrors(error: ValidationError): Record<string, string[]> {
  return error.issues.reduce<Record<string, string[]>>((fieldErrors, issue) => {
    const field = issue.path.map(String).join('.') || 'body';
    fieldErrors[field] = [...(fieldErrors[field] ?? []), issue.message];
    return fieldErrors;
  }, {});
}
