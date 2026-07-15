import 'server-only';

import type {
  AuthResponse,
  AuthUser,
  LoginRequest,
  RegisterRequest,
} from '@/features/auth/types';
import { authTokenMaxAgeSeconds } from '@/features/auth/server/auth-token';
import { ApiErrorCode, type ApiErrorCodeValue } from '@/http/codes';
import {
  GoApiClient,
  type GoApiError,
} from '@/server/api-adapter/go-api-client';

export interface GoApiAuthAdapterConfig {
  apiUrl: string;
  timeoutMs: number;
  maxResponseBytes?: number;
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
  private readonly dependencies: AdapterDependencies;
  private readonly client: GoApiClient;

  constructor(
    config: GoApiAuthAdapterConfig,
    dependencies: Partial<AdapterDependencies> = {}
  ) {
    this.dependencies = {
      fetch: dependencies.fetch ?? fetch,
      now: dependencies.now ?? Date.now,
      randomUUID: dependencies.randomUUID ?? (() => crypto.randomUUID()),
    };
    this.client = new GoApiClient(
      {
        apiUrl: config.apiUrl,
        timeoutMs: config.timeoutMs,
        maxResponseBytes: config.maxResponseBytes ?? 1_048_576,
        clientIpHeader: config.clientIpHeader,
      },
      {
        fetch: this.dependencies.fetch,
        randomUUID: this.dependencies.randomUUID,
      }
    );
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
    const result = await this.client.request({
      method: options.method,
      path,
      incomingHeaders,
      body: options.body,
      accessToken: options.accessToken,
      fieldMap: authFieldMap(operation),
      signal,
    });
    if (!result.ok) {
      return { ok: false, error: authenticationError(operation, result.error) };
    }
    if (!result.data.envelope) {
      return adapterUnavailable(result.data.requestId);
    }

    return {
      ok: true,
      data: {
        data: result.data.envelope.data,
        requestId: result.data.requestId,
      },
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

function authFieldMap(
  operation: AdapterOperation
): Readonly<Record<string, string>> {
  return operation === 'register'
    ? { email: 'email', nickname: 'name', password: 'password' }
    : operation === 'login'
      ? { username: 'email', password: 'password' }
      : {};
}

function authenticationError(
  operation: AdapterOperation,
  error: GoApiError
): AdapterError {
  const message = error.cause === 'timeout'
    ? 'Authentication service timed out'
    : error.cause === 'unavailable'
      ? 'Authentication service unavailable'
      : genericErrorMessage(operation, error.status);

  return {
    status: error.status,
    errorCode: error.errorCode,
    message,
    ...(error.fieldErrors ? { fieldErrors: error.fieldErrors } : {}),
    requestId: error.requestId,
    ...(error.responseHeaders ? { responseHeaders: error.responseHeaders } : {}),
  };
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

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0;
}
