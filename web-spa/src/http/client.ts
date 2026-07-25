import type { ZodType } from 'zod';
import { env } from '@/config/env';
import { ClientErrorCode, HttpStatusErrorCodeMap, isErrorCode } from './codes';

type ResponseMode = 'envelope' | 'json';
type ApiFieldErrors = Record<string, string[]>;

interface ApiSuccessEnvelope<T> {
  code: 0;
  message: string;
  data: T;
  meta?: unknown;
  links?: unknown;
}

interface RequestOptions<T> {
  headers?: HeadersInit;
  responseMode?: ResponseMode;
  schema?: ZodType<T>;
  signal?: AbortSignal;
}

export class ApiError extends Error {
  readonly errorCode: string;
  readonly fieldErrors?: ApiFieldErrors;
  readonly requestId?: string;
  readonly status?: number;

  constructor(
    message: string,
    errorCode: string,
    options: {
      fieldErrors?: ApiFieldErrors;
      requestId?: string;
      status?: number;
    } = {},
  ) {
    super(message);
    this.name = 'ApiError';
    this.errorCode = errorCode;
    if (options.fieldErrors !== undefined) {
      this.fieldErrors = options.fieldErrors;
    }
    if (options.requestId !== undefined) {
      this.requestId = options.requestId;
    }
    if (options.status !== undefined) {
      this.status = options.status;
    }
  }
}

function requestUrl(path: string): string {
  if (!path.startsWith('/') || path.startsWith('//')) {
    throw new Error('HTTP request paths must be root-relative');
  }
  return `${env.API_BASE_URL.replace(/\/$/, '')}${path}`;
}

function errorBody(value: unknown): {
  errorCode?: string;
  fieldErrors?: ApiFieldErrors;
  message?: string;
  requestId?: string;
} {
  if (!value || typeof value !== 'object') {
    return {};
  }

  const body = value as Record<string, unknown>;
  const result: {
    errorCode?: string;
    fieldErrors?: ApiFieldErrors;
    message?: string;
    requestId?: string;
  } = {};

  if (isErrorCode(body.error_code)) {
    result.errorCode = body.error_code;
  }
  if (typeof body.message === 'string' && body.message.trim()) {
    result.message = body.message;
  } else if (typeof body.error === 'string' && body.error.trim()) {
    result.message = body.error;
  }
  if (typeof body.request_id === 'string' && body.request_id.trim()) {
    result.requestId = body.request_id;
  }
  if (body.errors && typeof body.errors === 'object' && !Array.isArray(body.errors)) {
    const fields = Object.fromEntries(
      Object.entries(body.errors).filter(
        (entry): entry is [string, string[]] =>
          Array.isArray(entry[1]) && entry[1].every((item) => typeof item === 'string'),
      ),
    );
    if (Object.keys(fields).length > 0) {
      result.fieldErrors = fields;
    }
  }
  return result;
}

async function readJson(response: Response): Promise<unknown> {
  const declaredLength = Number(response.headers.get('content-length'));
  if (Number.isFinite(declaredLength) && declaredLength > env.API_MAX_RESPONSE_BYTES) {
    throw new ApiError(
      'Response exceeded the browser safety limit',
      ClientErrorCode.INVALID_RESPONSE,
      {
        status: response.status,
      },
    );
  }

  const text = await response.text();
  if (new TextEncoder().encode(text).byteLength > env.API_MAX_RESPONSE_BYTES) {
    throw new ApiError(
      'Response exceeded the browser safety limit',
      ClientErrorCode.INVALID_RESPONSE,
      {
        status: response.status,
      },
    );
  }
  if (!text) {
    return null;
  }

  const contentType = response.headers.get('content-type')?.toLowerCase() || '';
  if (!contentType.includes('json')) {
    throw new ApiError('API returned a non-JSON response', ClientErrorCode.INVALID_RESPONSE, {
      status: response.status,
    });
  }

  try {
    return JSON.parse(text) as unknown;
  } catch {
    throw new ApiError('API returned malformed JSON', ClientErrorCode.INVALID_RESPONSE, {
      status: response.status,
    });
  }
}

function parseSuccess<T>(
  payload: unknown,
  responseMode: ResponseMode,
  schema: ZodType<T> | undefined,
): T {
  let data = payload;

  if (responseMode === 'envelope') {
    if (
      !payload ||
      typeof payload !== 'object' ||
      (payload as Record<string, unknown>).code !== 0 ||
      !Object.prototype.hasOwnProperty.call(payload, 'data')
    ) {
      throw new ApiError(
        'API returned an invalid success envelope',
        ClientErrorCode.INVALID_RESPONSE,
      );
    }
    data = (payload as ApiSuccessEnvelope<unknown>).data;
  }

  if (!schema) {
    return data as T;
  }

  const parsed = schema.safeParse(data);
  if (!parsed.success) {
    throw new ApiError(
      'API response did not match the expected contract',
      ClientErrorCode.INVALID_RESPONSE,
    );
  }
  return parsed.data;
}

class HttpClient {
  async get<T>(path: string, options: RequestOptions<T> = {}): Promise<T> {
    return this.request(path, {
      ...options,
      method: 'GET',
    });
  }

  async post<T>(path: string, body?: unknown, options: RequestOptions<T> = {}): Promise<T> {
    return this.request(path, {
      ...options,
      body,
      method: 'POST',
    });
  }

  async put<T>(path: string, body?: unknown, options: RequestOptions<T> = {}): Promise<T> {
    return this.request(path, {
      ...options,
      body,
      method: 'PUT',
    });
  }

  async patch<T>(path: string, body?: unknown, options: RequestOptions<T> = {}): Promise<T> {
    return this.request(path, {
      ...options,
      body,
      method: 'PATCH',
    });
  }

  async delete<T>(path: string, options: RequestOptions<T> = {}): Promise<T> {
    return this.request(path, {
      ...options,
      method: 'DELETE',
    });
  }

  private async request<T>(
    path: string,
    options: RequestOptions<T> & {
      body?: unknown;
      method: 'DELETE' | 'GET' | 'PATCH' | 'POST' | 'PUT';
    },
  ): Promise<T> {
    const controller = new AbortController();
    let timedOut = false;
    const timeout = window.setTimeout(() => {
      timedOut = true;
      controller.abort();
    }, env.API_TIMEOUT_MS);
    const abort = () => controller.abort(options.signal?.reason);

    if (options.signal?.aborted) {
      abort();
    } else {
      options.signal?.addEventListener('abort', abort, { once: true });
    }

    const headers = new Headers(options.headers);
    headers.set('Accept', 'application/json');
    if (options.body !== undefined) {
      headers.set('Content-Type', 'application/json');
    }

    try {
      const init: RequestInit = {
        credentials: 'include',
        headers,
        method: options.method,
        signal: controller.signal,
      };
      if (options.body !== undefined) {
        init.body = JSON.stringify(options.body);
      }

      const response = await fetch(requestUrl(path), init);
      const payload = await readJson(response);

      if (!response.ok) {
        const body = errorBody(payload);
        const requestId = body.requestId || response.headers.get('x-request-id') || undefined;
        throw new ApiError(
          body.message || `Request failed with HTTP ${response.status}`,
          body.errorCode || HttpStatusErrorCodeMap[response.status] || ClientErrorCode.FETCH_ERROR,
          {
            ...(body.fieldErrors ? { fieldErrors: body.fieldErrors } : {}),
            ...(requestId ? { requestId } : {}),
            status: response.status,
          },
        );
      }

      return parseSuccess(payload, options.responseMode || 'envelope', options.schema);
    } catch (error) {
      if (error instanceof ApiError) {
        throw error;
      }
      if (controller.signal.aborted) {
        throw new ApiError(
          timedOut ? 'Request timed out' : 'Request was cancelled',
          timedOut ? ClientErrorCode.TIMEOUT : ClientErrorCode.CANCELLED,
        );
      }
      throw new ApiError('Unable to reach the API', ClientErrorCode.NETWORK_ERROR);
    } finally {
      window.clearTimeout(timeout);
      options.signal?.removeEventListener('abort', abort);
    }
  }
}

export const http = new HttpClient();
