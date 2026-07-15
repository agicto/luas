import 'server-only';

import { isIP } from 'node:net';

import { ApiErrorCode, HttpStatusErrorCodeMap, type ApiErrorCodeValue } from '@/http/codes';
import { declaredBodyLength, readBoundedBody } from '@/server/http/bounded-body';

const FORWARDED_RESPONSE_HEADERS = [
  'cache-control',
  'etag',
  'retry-after',
  'vary',
  'x-ratelimit-limit',
  'x-ratelimit-remaining',
  'x-ratelimit-reset',
] as const;
const SAFE_REQUEST_ID = /^[A-Za-z0-9._:-]{1,128}$/;
const SAFE_RELATIVE_PATH = /^[A-Za-z0-9._~!$&'()*+,;=@\/-]+$/;
const SAFE_IF_MATCH = /^[\x20-\x7E]{1,128}$/;
const SAFE_IF_NONE_MATCH = /^[\x20-\x7E]{1,1024}$/;
const knownErrorCodes = new Set<string>(Object.values(ApiErrorCode));

export interface GoApiClientConfig {
  apiUrl: string;
  timeoutMs: number;
  maxResponseBytes: number;
  clientIpHeader?: string;
}

interface GoApiClientDependencies {
  fetch: typeof fetch;
  randomUUID: () => string;
}

export type GoApiMethod = 'DELETE' | 'GET' | 'PATCH' | 'POST' | 'PUT';

export interface GoApiRequest {
  method: GoApiMethod;
  path: string;
  incomingHeaders: Headers;
  accessToken?: string;
  organizationId?: string;
  ifMatch?: string;
  ifNoneMatch?: string;
  body?: unknown;
  searchParams?: URLSearchParams;
  fieldMap?: Readonly<Record<string, string>>;
  signal?: AbortSignal;
}

export interface GoApiSuccessEnvelope {
  code: 0;
  message: string;
  data: unknown;
  meta?: unknown;
  links?: unknown;
}

export interface GoApiSuccess {
  status: number;
  envelope: GoApiSuccessEnvelope | null;
  requestId: string;
  responseHeaders?: Record<string, string>;
}

export interface GoApiError {
  cause: 'timeout' | 'unavailable' | 'upstream';
  status: number;
  errorCode: ApiErrorCodeValue;
  message: string;
  fieldErrors?: Record<string, string[]>;
  requestId: string;
  responseHeaders?: Record<string, string>;
}

export type GoApiResult = { ok: true; data: GoApiSuccess } | { ok: false; error: GoApiError };

export class GoApiClient {
  private readonly baseUrl: URL;
  private readonly dependencies: GoApiClientDependencies;

  constructor(
    private readonly config: GoApiClientConfig,
    dependencies: Partial<GoApiClientDependencies> = {}
  ) {
    this.baseUrl = new URL(`${config.apiUrl.replace(/\/+$/, '')}/`);
    this.dependencies = {
      fetch: dependencies.fetch ?? fetch,
      randomUUID: dependencies.randomUUID ?? (() => crypto.randomUUID()),
    };
  }

  request(options: GoApiRequest): Promise<GoApiResult> {
    if (!isSafeRelativePath(options.path)) {
      throw new Error('Go API requests require a fixed relative API path');
    }

    const url = new URL(options.path, this.baseUrl);
    if (options.searchParams) {
      url.search = options.searchParams.toString();
    }
    return this.performRequest(url, options);
  }

  private async performRequest(url: URL, options: GoApiRequest): Promise<GoApiResult> {
    const requestId = resolveRequestId(options.incomingHeaders, this.dependencies.randomUUID);
    const headers = new Headers({
      accept: 'application/json',
      'x-request-id': requestId,
    });

    if (options.body !== undefined) {
      headers.set('content-type', 'application/json');
    }
    if (options.accessToken) {
      headers.set('authorization', `Bearer ${options.accessToken}`);
    }
    if (options.organizationId) {
      headers.set('organization-id', options.organizationId);
    }
    if (options.ifMatch !== undefined) {
      if (!SAFE_IF_MATCH.test(options.ifMatch)) {
        throw new Error('Go API If-Match must be bounded visible ASCII');
      }
      headers.set('if-match', options.ifMatch);
    }
    if (options.ifNoneMatch !== undefined) {
      if (!SAFE_IF_NONE_MATCH.test(options.ifNoneMatch)) {
        throw new Error('Go API If-None-Match must be bounded visible ASCII');
      }
      headers.set('if-none-match', options.ifNoneMatch);
    }

    const clientIp = resolveClientIp(options.incomingHeaders, this.config.clientIpHeader);
    if (clientIp) {
      headers.set('x-forwarded-for', clientIp);
    }

    const timeoutSignal = AbortSignal.timeout(this.config.timeoutMs);
    const combinedSignal = options.signal
      ? AbortSignal.any([options.signal, timeoutSignal])
      : timeoutSignal;

    let response: Response;
    try {
      response = await this.dependencies.fetch(url, {
        method: options.method,
        headers,
        body: options.body === undefined ? undefined : JSON.stringify(options.body),
        cache: 'no-store',
        redirect: 'error',
        signal: combinedSignal,
      });
    } catch (error) {
      return isAbortError(error) || combinedSignal.aborted
        ? failure('timeout', ApiErrorCode.COMMON_TIMEOUT, requestId)
        : failure('unavailable', ApiErrorCode.COMMON_SERVICE_UNAVAILABLE, requestId);
    }

    const responseHeaders = forwardedResponseHeaders(response.headers);
    const responseRequestId = resolveResponseRequestId(response.headers, requestId);

    if (response.status === 204 || response.status === 304) {
      if (!response.ok && response.status !== 304) {
        return failure(
          'unavailable',
          ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
          responseRequestId,
          responseHeaders
        );
      }
      return {
        ok: true,
        data: {
          status: response.status,
          envelope: null,
          requestId: responseRequestId,
          ...(responseHeaders ? { responseHeaders } : {}),
        },
      };
    }

    const declaredLength = declaredBodyLength(response.headers, this.config.maxResponseBytes);
    if (declaredLength !== 'accepted') {
      await response.body?.cancel();
      return failure(
        'unavailable',
        ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
        responseRequestId,
        responseHeaders
      );
    }

    const body = await readBoundedBody(response.body, this.config.maxResponseBytes);
    if (!body.ok) {
      return failure(
        'unavailable',
        ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
        responseRequestId,
        responseHeaders
      );
    }

    const payload = parseJson(body.text);
    if (!response.ok) {
      return {
        ok: false,
        error: upstreamError(
          response,
          payload,
          responseRequestId,
          responseHeaders,
          options.fieldMap
        ),
      };
    }

    if (!isSuccessEnvelope(payload)) {
      return failure(
        'unavailable',
        ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
        responseRequestId,
        responseHeaders
      );
    }

    return {
      ok: true,
      data: {
        status: response.status,
        envelope: payload,
        requestId: responseRequestId,
        ...(responseHeaders ? { responseHeaders } : {}),
      },
    };
  }
}

function isSafeRelativePath(path: string): boolean {
  if (
    path.length === 0 ||
    path.startsWith('/') ||
    !SAFE_RELATIVE_PATH.test(path) ||
    /^[A-Za-z][A-Za-z0-9+.-]*:/.test(path)
  ) {
    return false;
  }

  return path.split('/').every(segment => segment !== '.' && segment !== '..');
}

function upstreamError(
  response: Response,
  payload: unknown,
  fallbackRequestId: string,
  responseHeaders: Record<string, string> | undefined,
  fieldMap: Readonly<Record<string, string>> | undefined
): GoApiError {
  const body = isRecord(payload) ? payload : {};
  const status = response.status >= 400 && response.status <= 599 ? response.status : 503;
  const candidateCode = body.error_code;
  const errorCode =
    typeof candidateCode === 'string' && knownErrorCodes.has(candidateCode)
      ? (candidateCode as ApiErrorCodeValue)
      : (HttpStatusErrorCodeMap[status] ?? ApiErrorCode.COMMON_SERVICE_UNAVAILABLE);
  const requestId =
    typeof body.request_id === 'string' && SAFE_REQUEST_ID.test(body.request_id)
      ? body.request_id
      : fallbackRequestId;
  const fieldErrors = mapFieldErrors(body.errors, fieldMap);

  return {
    cause: 'upstream',
    status,
    errorCode,
    message: 'Upstream API request failed',
    ...(fieldErrors ? { fieldErrors } : {}),
    requestId,
    ...(responseHeaders ? { responseHeaders } : {}),
  };
}

function failure(
  cause: 'timeout' | 'unavailable',
  errorCode: ApiErrorCodeValue,
  requestId: string,
  responseHeaders?: Record<string, string>
): GoApiResult {
  return {
    ok: false,
    error: {
      cause,
      status: 503,
      errorCode,
      message: cause === 'timeout' ? 'Upstream API timed out' : 'Upstream API unavailable',
      requestId,
      ...(responseHeaders ? { responseHeaders } : {}),
    },
  };
}

function mapFieldErrors(
  value: unknown,
  fieldMap: Readonly<Record<string, string>> | undefined
): Record<string, string[]> | undefined {
  if (!fieldMap || !isRecord(value)) {
    return undefined;
  }

  const mapped: Record<string, string[]> = {};
  for (const [upstreamField, browserField] of Object.entries(fieldMap)) {
    if (Array.isArray(value[upstreamField]) && value[upstreamField].length > 0) {
      mapped[browserField] = ['Invalid value'];
    }
  }
  return Object.keys(mapped).length > 0 ? mapped : undefined;
}

function resolveRequestId(headers: Headers, randomUUID: () => string): string {
  const candidate = headers.get('x-request-id');
  return candidate && SAFE_REQUEST_ID.test(candidate) ? candidate : randomUUID();
}

function resolveResponseRequestId(headers: Headers, fallback: string): string {
  const candidate = headers.get('x-request-id');
  return candidate && SAFE_REQUEST_ID.test(candidate) ? candidate : fallback;
}

function resolveClientIp(headers: Headers, sourceHeader: string | undefined): string | undefined {
  if (!sourceHeader) {
    return undefined;
  }
  const candidate = headers.get(sourceHeader)?.trim();
  if (!candidate || candidate.includes(',')) {
    return undefined;
  }
  return isIP(candidate) !== 0 ? candidate : undefined;
}

function forwardedResponseHeaders(headers: Headers): Record<string, string> | undefined {
  const forwarded: Record<string, string> = {};
  for (const name of FORWARDED_RESPONSE_HEADERS) {
    const value = headers.get(name);
    if (value) {
      forwarded[name] = value;
    }
  }
  return Object.keys(forwarded).length > 0 ? forwarded : undefined;
}

function parseJson(value: string): unknown {
  try {
    return JSON.parse(value) as unknown;
  } catch {
    return null;
  }
}

function isSuccessEnvelope(value: unknown): value is GoApiSuccessEnvelope {
  return (
    isRecord(value) &&
    value.code === 0 &&
    typeof value.message === 'string' &&
    Object.prototype.hasOwnProperty.call(value, 'data')
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function isAbortError(error: unknown): boolean {
  return error instanceof Error && (error.name === 'AbortError' || error.name === 'TimeoutError');
}
