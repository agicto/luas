import 'server-only';

import { isIP } from 'node:net';

import type {
  AuthResponse,
  AuthUser,
  LoginRequest,
  RegisterRequest,
} from '@/features/auth/types';
import { authTokenMaxAgeSeconds } from '@/features/auth/server/auth-token';
import {
  ApiErrorCode,
  HttpStatusErrorCodeMap,
  type ApiErrorCodeValue,
} from '@/http/codes';

const FORWARDED_RESPONSE_HEADERS = [
  'retry-after',
  'x-ratelimit-limit',
  'x-ratelimit-remaining',
  'x-ratelimit-reset',
] as const;
const SAFE_REQUEST_ID = /^[A-Za-z0-9._:-]{1,128}$/;
const knownErrorCodes = new Set<string>(Object.values(ApiErrorCode));

export interface GoApiAuthAdapterConfig {
  apiUrl: string;
  timeoutMs: number;
  clientIpHeader?: string;
}

interface AdapterDependencies {
  fetch: typeof fetch;
  now: () => number;
  randomUUID: () => string;
}

export interface AdapterError {
  status: number;
  errorCode: ApiErrorCodeValue;
  message: string;
  fieldErrors?: Record<string, string[]>;
  requestId?: string;
  responseHeaders?: Record<string, string>;
}

export type AdapterResult<T> =
  | { ok: true; data: T }
  | { ok: false; error: AdapterError };

export interface AdapterSession extends AuthResponse {
  accessToken: string;
  maxAgeSeconds: number;
}

type AdapterOperation = 'login' | 'register' | 'session';

interface GoApiUser {
  id: number;
  username: string;
  email: string;
  nickname?: string;
  status: number;
}

interface UpstreamSuccess {
  data: unknown;
  requestId: string;
}

export class GoApiAuthAdapter {
  private readonly baseUrl: URL;
  private readonly dependencies: AdapterDependencies;

  constructor(
    private readonly config: GoApiAuthAdapterConfig,
    dependencies: Partial<AdapterDependencies> = {}
  ) {
    this.baseUrl = new URL(`${config.apiUrl.replace(/\/+$/, '')}/`);
    this.dependencies = {
      fetch: dependencies.fetch ?? fetch,
      now: dependencies.now ?? Date.now,
      randomUUID: dependencies.randomUUID ?? (() => crypto.randomUUID()),
    };
  }

  async login(
    input: LoginRequest,
    incomingHeaders: Headers,
    signal?: AbortSignal
  ): Promise<AdapterResult<AdapterSession>> {
    const result = await this.request(
      'login',
      'login',
      {
        method: 'POST',
        body: {
          username: input.email,
          password: input.password,
        },
      },
      incomingHeaders,
      signal
    );

    if (!result.ok) {
      return result;
    }

    return this.sessionFromLoginData(result.data.data, result.data.requestId);
  }

  async register(
    input: RegisterRequest,
    incomingHeaders: Headers,
    signal?: AbortSignal
  ): Promise<AdapterResult<AdapterSession>> {
    const registration = await this.request(
      'register',
      'register',
      {
        method: 'POST',
        body: {
          username: generatedUsername(this.dependencies.randomUUID()),
          password: input.password,
          email: input.email,
          nickname: input.name,
        },
      },
      incomingHeaders,
      signal
    );

    if (!registration.ok) {
      return registration;
    }

    if (!mapGoApiUser(registration.data.data)) {
      return adapterUnavailable(registration.data.requestId);
    }

    return this.login(
      { email: input.email, password: input.password },
      incomingHeaders,
      signal
    );
  }

  async profile(
    accessToken: string,
    incomingHeaders: Headers,
    signal?: AbortSignal
  ): Promise<AdapterResult<AuthResponse>> {
    const result = await this.request(
      'session',
      'users/profile',
      {
        method: 'GET',
        accessToken,
      },
      incomingHeaders,
      signal
    );

    if (!result.ok) {
      return result;
    }

    const user = mapGoApiUser(result.data.data);
    if (!user) {
      return adapterUnavailable(result.data.requestId);
    }
    if (!isActiveGoApiUser(result.data.data)) {
      return accountDisabled(result.data.requestId);
    }

    return { ok: true, data: { user } };
  }

  private sessionFromLoginData(
    value: unknown,
    requestId: string
  ): AdapterResult<AdapterSession> {
    if (!isRecord(value)) {
      return adapterUnavailable(requestId);
    }

    const user = mapGoApiUser(value.user);
    const maxAgeSeconds =
      typeof value.access_token === 'string'
        ? authTokenMaxAgeSeconds(value.access_token, this.dependencies.now())
        : null;

    if (!user || !maxAgeSeconds) {
      return adapterUnavailable(requestId);
    }
    if (!isActiveGoApiUser(value.user)) {
      return accountDisabled(requestId);
    }

    return {
      ok: true,
      data: {
        user,
        accessToken: value.access_token as string,
        maxAgeSeconds,
      },
    };
  }

  private async request(
    operation: AdapterOperation,
    path: string,
    options: {
      method: 'GET' | 'POST';
      body?: Record<string, string>;
      accessToken?: string;
    },
    incomingHeaders: Headers,
    signal?: AbortSignal
  ): Promise<AdapterResult<UpstreamSuccess>> {
    const requestId = resolveRequestId(incomingHeaders, this.dependencies.randomUUID);
    const headers = new Headers({
      accept: 'application/json',
      'x-request-id': requestId,
    });

    if (options.body) {
      headers.set('content-type', 'application/json');
    }
    if (options.accessToken) {
      headers.set('authorization', `Bearer ${options.accessToken}`);
    }

    const clientIp = resolveClientIp(incomingHeaders, this.config.clientIpHeader);
    if (clientIp) {
      headers.set('x-forwarded-for', clientIp);
    }

    const timeoutSignal = AbortSignal.timeout(this.config.timeoutMs);
    const combinedSignal = signal
      ? AbortSignal.any([signal, timeoutSignal])
      : timeoutSignal;

    let response: Response;
    try {
      response = await this.dependencies.fetch(new URL(path, this.baseUrl), {
        method: options.method,
        headers,
        body: options.body ? JSON.stringify(options.body) : undefined,
        cache: 'no-store',
        redirect: 'error',
        signal: combinedSignal,
      });
    } catch (error) {
      return isAbortError(error) || combinedSignal.aborted
        ? adapterTimeout(requestId)
        : adapterUnavailable(requestId);
    }

    const payload = await readJson(response);
    if (!response.ok) {
      return {
        ok: false,
        error: upstreamError(operation, response, payload, requestId),
      };
    }

    if (!isRecord(payload) || payload.code !== 0 || !('data' in payload)) {
      return adapterUnavailable(requestId);
    }

    return {
      ok: true,
      data: { data: payload.data, requestId },
    };
  }
}

export function generatedUsername(randomId: string): string {
  const suffix = randomId.toLowerCase().replace(/[^a-z0-9]/g, '').slice(0, 32);

  if (suffix.length < 16) {
    throw new Error('Generated auth username requires at least 16 random characters');
  }

  return `user_${suffix}`;
}

export function mapGoApiUser(value: unknown): AuthUser | null {
  if (!isGoApiUser(value)) {
    return null;
  }

  return {
    id: String(value.id),
    email: value.email,
    name: value.nickname?.trim() || value.username,
  };
}

function isGoApiUser(value: unknown): value is GoApiUser {
  if (!isRecord(value)) {
    return false;
  }

  return (
    typeof value.id === 'number' &&
    Number.isSafeInteger(value.id) &&
    value.id > 0 &&
    isNonEmptyString(value.username) &&
    isNonEmptyString(value.email) &&
    (value.nickname === undefined || typeof value.nickname === 'string') &&
    typeof value.status === 'number'
  );
}

function isActiveGoApiUser(value: unknown): boolean {
  return isGoApiUser(value) && value.status === 1;
}

function resolveRequestId(
  headers: Headers,
  randomUUID: () => string
): string {
  const candidate = headers.get('x-request-id');
  return candidate && SAFE_REQUEST_ID.test(candidate) ? candidate : randomUUID();
}

function resolveClientIp(
  headers: Headers,
  sourceHeader: string | undefined
): string | undefined {
  if (!sourceHeader) {
    return undefined;
  }

  const candidate = headers.get(sourceHeader)?.trim();

  if (!candidate || candidate.includes(',')) {
    return undefined;
  }

  return isIP(candidate) !== 0 ? candidate : undefined;
}

function upstreamError(
  operation: AdapterOperation,
  response: Response,
  payload: unknown,
  fallbackRequestId: string
): AdapterError {
  const body = isRecord(payload) ? payload : {};
  const status = response.status >= 400 && response.status <= 599
    ? response.status
    : 503;
  const candidateCode = body.error_code;
  const errorCode =
    typeof candidateCode === 'string' && knownErrorCodes.has(candidateCode)
      ? (candidateCode as ApiErrorCodeValue)
      : HttpStatusErrorCodeMap[status] ?? ApiErrorCode.COMMON_SERVICE_UNAVAILABLE;
  const requestId =
    typeof body.request_id === 'string' && SAFE_REQUEST_ID.test(body.request_id)
      ? body.request_id
      : fallbackRequestId;
  const fieldErrors = mapFieldErrors(operation, body.errors);
  const responseHeaders = forwardedResponseHeaders(response.headers);

  return {
    status,
    errorCode,
    message: genericErrorMessage(operation, status),
    ...(fieldErrors ? { fieldErrors } : {}),
    requestId,
    ...(responseHeaders ? { responseHeaders } : {}),
  };
}

function mapFieldErrors(
  operation: AdapterOperation,
  value: unknown
): Record<string, string[]> | undefined {
  if (!isRecord(value)) {
    return undefined;
  }

  const allowedFields = operation === 'register'
    ? { email: 'email', nickname: 'name', password: 'password' }
    : operation === 'login'
      ? { username: 'email', password: 'password' }
      : {};
  const mapped: Record<string, string[]> = {};

  for (const [upstreamField, browserField] of Object.entries(allowedFields)) {
    if (Array.isArray(value[upstreamField]) && value[upstreamField].length > 0) {
      mapped[browserField] = ['Invalid value'];
    }
  }

  return Object.keys(mapped).length > 0 ? mapped : undefined;
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

function genericErrorMessage(operation: AdapterOperation, status: number): string {
  if (status === 401) {
    return operation === 'session' ? 'Authentication required' : 'Authentication failed';
  }
  if (status === 403) {
    return 'Access forbidden';
  }
  if (status === 409) {
    return 'Registration conflict';
  }
  if (status === 400 || status === 422) {
    return 'Invalid authentication input';
  }
  if (status === 429) {
    return 'Too many authentication attempts';
  }
  return 'Authentication service unavailable';
}

function adapterTimeout(requestId: string): AdapterResult<never> {
  return {
    ok: false,
    error: {
      status: 503,
      errorCode: ApiErrorCode.COMMON_TIMEOUT,
      message: 'Authentication service timed out',
      requestId,
    },
  };
}

function accountDisabled(requestId: string): AdapterResult<never> {
  return {
    ok: false,
    error: {
      status: 403,
      errorCode: ApiErrorCode.AUTH_ACCOUNT_DISABLED,
      message: 'Account access is disabled',
      requestId,
    },
  };
}

function adapterUnavailable(requestId: string): AdapterResult<never> {
  return {
    ok: false,
    error: {
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message: 'Authentication service unavailable',
      requestId,
    },
  };
}

function isAbortError(error: unknown): boolean {
  return (
    error instanceof Error &&
    (error.name === 'AbortError' || error.name === 'TimeoutError')
  );
}

async function readJson(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return null;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0;
}
