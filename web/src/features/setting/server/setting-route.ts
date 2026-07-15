import 'server-only';

import { NextResponse } from 'next/server';

import { apiErrorResponse, apiJsonBodyErrorResponse } from '@/app/api/_shared/error-response';
import { readJsonBody } from '@/app/api/_shared/json-body';
import { guardSameOriginMutation } from '@/app/api/_shared/mock-bff';
import { apiNoContentResponse, apiSuccessResponse } from '@/app/api/_shared/success-response';
import { env } from '@/config/env';
import { isWebFeatureEnabled } from '@/config/features';
import { isMockBffEnabled } from '@/config/mock-bff';
import { serverEnv } from '@/config/server-env';
import { getApiSessionToken } from '@/features/auth/server/api-session';
import { getSessionUser } from '@/features/auth/server/session';
import type { AuthUser } from '@/features/auth/types';
import { mockOrganizationStore } from '@/features/organization/server/mock-organization-store';
import { parseOrganizationId } from '@/features/organization/server/organization-route';
import type { OrganizationContext } from '@/features/organization/types';
import {
  localeSettingMutationSchema,
  organizationSettingRouteKeySchema,
  timezoneSettingMutationSchema,
  userSettingRouteKeySchema,
} from '@/features/setting/schemas';
import {
  mockSettingStore,
  type MockSettingMutationResult,
} from '@/features/setting/server/mock-setting-store';
import type {
  OrganizationSettingKey,
  SettingMutation,
  UserSettingKey,
} from '@/features/setting/types';
import { ApiErrorCode } from '@/http/codes';
import { forwardAuthenticatedGoApi } from '@/server/api-adapter/authenticated-route';
import { forwardPublicGoApi } from '@/server/api-adapter/public-route';
import { privateNoStoreHeaders, privateNoStoreResponse } from '@/server/http/private-response';

type SettingBackend = 'go-api' | 'mock';

interface SettingRouteEnvironment {
  adapterEnabled: boolean;
  featureEnabled: boolean;
  mockBffEnabled: boolean;
  nodeEnv: 'development' | 'production' | 'test';
}

type SettingRouteResolution =
  { available: true; backend: SettingBackend } | { available: false; response: NextResponse };

type AuthenticatedSettingBackend =
  | { authenticated: true; backend: 'go-api'; accessToken: string }
  | { authenticated: true; backend: 'mock'; user: AuthUser }
  | { authenticated: false; response: NextResponse };

type OrganizationSettingTarget =
  | {
      available: true;
      organizationId: number;
      authentication: Extract<AuthenticatedSettingBackend, { authenticated: true }>;
      context?: OrganizationContext;
    }
  | { available: false; response: NextResponse };

const settingVersionPattern = /^"setting-v(0|[1-9]\d*)"$/u;
const maxSettingMutationBytes = 4 * 1_024 + 256;

export function resolveSettingRoute(
  environment: SettingRouteEnvironment = currentEnvironment()
): SettingRouteResolution {
  if (!environment.featureEnabled) {
    return unavailable('Setting Web feature is disabled');
  }
  if (environment.adapterEnabled) return { available: true, backend: 'go-api' };
  if (
    isMockBffEnabled({
      enabled: environment.mockBffEnabled,
      nodeEnv: environment.nodeEnv,
    })
  ) {
    return { available: true, backend: 'mock' };
  }
  return unavailable('Setting backend is unavailable');
}

export function privateSettingResponse<T extends Response>(response: T, organization = false): T {
  return privateNoStoreResponse(
    response,
    organization ? ['Cookie', 'Organization-Id'] : ['Cookie']
  );
}

export async function publicSettingsRoute(request: Request): Promise<NextResponse> {
  const resolution = resolveSettingRoute();
  if (!resolution.available) return resolution.response;
  const validator = request.headers.get('if-none-match');
  if (validator !== null && !safeConditionalHeader(validator, 1_024)) {
    return invalidConditionalHeader(publicNoStoreHeaders());
  }
  if (resolution.backend === 'go-api') {
    return forwardPublicGoApi(request, {
      method: 'GET',
      path: 'settings/public',
      ...(validator === null ? {} : { ifNoneMatch: validator }),
    });
  }

  const etag = mockSettingStore.publicETag();
  const headers = publicSettingHeaders(etag);
  if (validator !== null && matchesETag(validator, etag)) {
    return new NextResponse(null, { status: 304, headers });
  }
  return apiSuccessResponse(mockSettingStore.publicApp(), { headers });
}

export async function userSettingsRoute(request: Request): Promise<NextResponse> {
  const route = await resolveAuthenticatedSettingRoute();
  if (!route.available) return route.response;
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'settings/user',
      accessToken: route.authentication.accessToken,
    });
  }
  return apiSuccessResponse(mockSettingStore.user(route.authentication.user), {
    headers: privateSettingHeaders(false),
  });
}

export async function setUserSettingRoute(request: Request, rawKey: string): Promise<NextResponse> {
  const route = await resolveSettingMutationRoute(request);
  if (!route.available) return route.response;
  const key = parseUserSettingKey(rawKey);
  if (!key) return settingNotFound(privateSettingHeaders(false));
  const expectedVersion = expectedSettingVersion(request, privateSettingHeaders(false));
  if (!expectedVersion.ok) return expectedVersion.response;
  const input = await settingMutationBody(request, key, false);
  if (!input.ok) return input.response;

  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'PATCH',
      path: `settings/user/${key}`,
      accessToken: route.authentication.accessToken,
      ifMatch: expectedVersion.etag,
      body: input.value,
      fieldMap: { value: 'value' },
    });
  }
  return mockMutationResponse(
    mockSettingStore.setUser(route.authentication.user, key, input.value, expectedVersion.version),
    false
  );
}

export async function resetUserSettingRoute(
  request: Request,
  rawKey: string
): Promise<NextResponse> {
  const route = await resolveSettingMutationRoute(request);
  if (!route.available) return route.response;
  const key = parseUserSettingKey(rawKey);
  if (!key) return settingNotFound(privateSettingHeaders(false));
  const expectedVersion = expectedSettingVersion(request, privateSettingHeaders(false));
  if (!expectedVersion.ok) return expectedVersion.response;
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'DELETE',
      path: `settings/user/${key}`,
      accessToken: route.authentication.accessToken,
      ifMatch: expectedVersion.etag,
    });
  }
  return mockResetResponse(
    mockSettingStore.resetUser(route.authentication.user, key, expectedVersion.version),
    false
  );
}

export async function organizationSettingsRoute(request: Request): Promise<NextResponse> {
  const target = await resolveOrganizationSettingTarget(request);
  if (!target.available) return target.response;
  if (target.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'organization-settings',
      accessToken: target.authentication.accessToken,
      organizationId: String(target.organizationId),
    });
  }
  return apiSuccessResponse(mockSettingStore.organization(target.organizationId), {
    headers: privateSettingHeaders(true),
  });
}

export async function setOrganizationSettingRoute(
  request: Request,
  rawKey: string
): Promise<NextResponse> {
  const target = await resolveOrganizationSettingMutationTarget(request);
  if (!target.available) return target.response;
  const key = parseOrganizationSettingKey(rawKey);
  if (!key) return settingNotFound(privateSettingHeaders(true));
  if (
    target.authentication.backend === 'mock' &&
    target.context?.role !== 'owner' &&
    target.context?.role !== 'admin'
  ) {
    return forbidden(privateSettingHeaders(true));
  }
  const expectedVersion = expectedSettingVersion(request, privateSettingHeaders(true));
  if (!expectedVersion.ok) return expectedVersion.response;
  const input = await settingMutationBody(request, key, true);
  if (!input.ok) return input.response;

  if (target.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'PATCH',
      path: `organization-settings/${key}`,
      accessToken: target.authentication.accessToken,
      organizationId: String(target.organizationId),
      ifMatch: expectedVersion.etag,
      body: input.value,
      fieldMap: { value: 'value' },
    });
  }
  return mockMutationResponse(
    mockSettingStore.setOrganization(
      target.organizationId,
      key,
      input.value,
      expectedVersion.version
    ),
    true
  );
}

export async function resetOrganizationSettingRoute(
  request: Request,
  rawKey: string
): Promise<NextResponse> {
  const target = await resolveOrganizationSettingMutationTarget(request);
  if (!target.available) return target.response;
  const key = parseOrganizationSettingKey(rawKey);
  if (!key) return settingNotFound(privateSettingHeaders(true));
  if (
    target.authentication.backend === 'mock' &&
    target.context?.role !== 'owner' &&
    target.context?.role !== 'admin'
  ) {
    return forbidden(privateSettingHeaders(true));
  }
  const expectedVersion = expectedSettingVersion(request, privateSettingHeaders(true));
  if (!expectedVersion.ok) return expectedVersion.response;
  if (target.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'DELETE',
      path: `organization-settings/${key}`,
      accessToken: target.authentication.accessToken,
      organizationId: String(target.organizationId),
      ifMatch: expectedVersion.etag,
    });
  }
  return mockResetResponse(
    mockSettingStore.resetOrganization(target.organizationId, key, expectedVersion.version),
    true
  );
}

async function resolveSettingMutationRoute(request: Request) {
  const resolution = resolveSettingRoute();
  if (!resolution.available) return resolution;
  const guard = guardSameOriginMutation(request);
  if (guard) {
    return { available: false as const, response: privateSettingResponse(guard, false) };
  }
  return resolveAuthenticatedSettingRoute(resolution);
}

async function resolveAuthenticatedSettingRoute(
  resolution: SettingRouteResolution = resolveSettingRoute()
) {
  if (!resolution.available) return resolution;
  const authentication = await authenticateSettingBackend(resolution.backend);
  return authentication.authenticated
    ? { available: true as const, authentication }
    : { available: false as const, response: authentication.response };
}

async function authenticateSettingBackend(
  backend: SettingBackend
): Promise<AuthenticatedSettingBackend> {
  if (backend === 'go-api') {
    const accessToken = await getApiSessionToken();
    return accessToken
      ? { authenticated: true, backend, accessToken }
      : { authenticated: false, response: unauthenticated(privateSettingHeaders(false)) };
  }
  const user = await getSessionUser();
  return user
    ? { authenticated: true, backend, user }
    : { authenticated: false, response: unauthenticated(privateSettingHeaders(false)) };
}

async function resolveOrganizationSettingTarget(
  request: Request
): Promise<OrganizationSettingTarget> {
  const route = await resolveAuthenticatedSettingRoute();
  if (!route.available) return route;
  return resolveOrganizationTargetFromAuthentication(route.authentication, request);
}

async function resolveOrganizationSettingMutationTarget(
  request: Request
): Promise<OrganizationSettingTarget> {
  const resolution = resolveSettingRoute();
  if (!resolution.available) return resolution;
  const guard = guardSameOriginMutation(request);
  if (guard) {
    return { available: false, response: privateSettingResponse(guard, true) };
  }
  const route = await resolveAuthenticatedSettingRoute(resolution);
  if (!route.available) return route;
  return resolveOrganizationTargetFromAuthentication(route.authentication, request);
}

function resolveOrganizationTargetFromAuthentication(
  authentication: Extract<AuthenticatedSettingBackend, { authenticated: true }>,
  request: Request
): OrganizationSettingTarget {
  const rawOrganizationId = request.headers.get('organization-id');
  if (rawOrganizationId === null) return organizationContextRequired();
  const organizationId = parseOrganizationId(rawOrganizationId);
  if (!organizationId) return organizationContextInvalid();
  if (authentication.backend === 'go-api') {
    return { available: true, organizationId, authentication };
  }
  const context = mockOrganizationStore.resolveContext(authentication.user, organizationId);
  if (!context) {
    return {
      available: false,
      response: apiErrorResponse({
        status: 404,
        errorCode: ApiErrorCode.ORGANIZATION_NOT_FOUND,
        message: 'Organization not found',
        headers: privateSettingHeaders(true),
      }),
    };
  }
  return { available: true, organizationId, authentication, context };
}

async function settingMutationBody(
  request: Request,
  key: UserSettingKey | OrganizationSettingKey,
  organization: boolean
): Promise<{ ok: true; value: SettingMutation } | { ok: false; response: NextResponse }> {
  const payload = await readJsonBody(request, maxSettingMutationBytes);
  if (!payload.ok) {
    return {
      ok: false,
      response: privateSettingResponse(apiJsonBodyErrorResponse(payload.error), organization),
    };
  }
  const schema =
    key === 'localization.timezone' ? timezoneSettingMutationSchema : localeSettingMutationSchema;
  const parsed = schema.safeParse(payload.data);
  if (!parsed.success) {
    return {
      ok: false,
      response: apiErrorResponse({
        status: 422,
        errorCode: ApiErrorCode.SETTING_INVALID_VALUE,
        message: 'Invalid setting value',
        errors: { value: ['Invalid value'] },
        headers: privateSettingHeaders(organization),
      }),
    };
  }
  return { ok: true, value: parsed.data };
}

function expectedSettingVersion(
  request: Request,
  headers: Headers
): { ok: true; version: number; etag: string } | { ok: false; response: NextResponse } {
  const etag = request.headers.get('if-match');
  if (etag === null) {
    return {
      ok: false,
      response: apiErrorResponse({
        status: 428,
        errorCode: ApiErrorCode.SETTING_PRECONDITION_REQUIRED,
        message: 'Setting version precondition required',
        headers,
      }),
    };
  }
  const match = settingVersionPattern.exec(etag);
  if (!match || etag.includes(',')) {
    return { ok: false, response: invalidConditionalHeader(headers) };
  }
  const version = Number(match[1]);
  if (!Number.isSafeInteger(version)) {
    return { ok: false, response: invalidConditionalHeader(headers) };
  }
  return { ok: true, version, etag };
}

function parseUserSettingKey(value: string): UserSettingKey | null {
  const parsed = userSettingRouteKeySchema.safeParse(value);
  return parsed.success ? parsed.data : null;
}

function parseOrganizationSettingKey(value: string): OrganizationSettingKey | null {
  const parsed = organizationSettingRouteKeySchema.safeParse(value);
  return parsed.success ? parsed.data : null;
}

function mockMutationResponse(
  result: MockSettingMutationResult,
  organization: boolean
): NextResponse {
  if (!result.ok) return mockSettingError(result.error, privateSettingHeaders(organization));
  const headers = privateSettingHeaders(organization);
  headers.set('etag', `"setting-v${result.setting.version}"`);
  return apiSuccessResponse(result.setting, { headers });
}

function mockResetResponse(result: MockSettingMutationResult, organization: boolean): NextResponse {
  if (!result.ok) return mockSettingError(result.error, privateSettingHeaders(organization));
  const headers = privateSettingHeaders(organization);
  headers.set('etag', `"setting-v${result.setting.version}"`);
  return apiNoContentResponse(headers);
}

function mockSettingError(error: 'not_found' | 'version_conflict', headers: Headers): NextResponse {
  return error === 'not_found'
    ? settingNotFound(headers)
    : apiErrorResponse({
        status: 412,
        errorCode: ApiErrorCode.SETTING_VERSION_CONFLICT,
        message: 'Setting version conflict',
        headers,
      });
}

function currentEnvironment(): SettingRouteEnvironment {
  return {
    adapterEnabled: serverEnv.API_ADAPTER_ENABLED,
    featureEnabled: isWebFeatureEnabled('setting'),
    mockBffEnabled: serverEnv.MOCK_BFF_ENABLED,
    nodeEnv: env.NODE_ENV,
  };
}

function privateSettingHeaders(organization: boolean): Headers {
  return privateNoStoreHeaders(
    undefined,
    organization ? ['Cookie', 'Organization-Id'] : ['Cookie']
  );
}

function publicSettingHeaders(etag: string): Headers {
  return new Headers({
    'cache-control': 'public, max-age=60, stale-while-revalidate=300',
    etag,
  });
}

function publicNoStoreHeaders(): Headers {
  return new Headers({ 'cache-control': 'no-store', pragma: 'no-cache' });
}

function safeConditionalHeader(value: string, maximum: number): boolean {
  return value.length >= 1 && value.length <= maximum && /^[\x20-\x7E]+$/u.test(value);
}

function matchesETag(header: string, etag: string): boolean {
  return header.split(',').some(candidate => {
    const normalized = candidate.trim();
    return normalized === '*' || normalized === etag || normalized.replace(/^W\//u, '') === etag;
  });
}

function invalidConditionalHeader(headers: Headers): NextResponse {
  return apiErrorResponse({
    status: 400,
    errorCode: ApiErrorCode.COMMON_INVALID_INPUT,
    message: 'Invalid conditional request header',
    headers,
  });
}

function settingNotFound(headers: Headers): NextResponse {
  return apiErrorResponse({
    status: 404,
    errorCode: ApiErrorCode.SETTING_NOT_FOUND,
    message: 'Setting not found',
    headers,
  });
}

function organizationContextRequired(): OrganizationSettingTarget {
  return {
    available: false,
    response: apiErrorResponse({
      status: 400,
      errorCode: ApiErrorCode.ORGANIZATION_CONTEXT_REQUIRED,
      message: 'Organization context is required',
      headers: privateSettingHeaders(true),
    }),
  };
}

function organizationContextInvalid(): OrganizationSettingTarget {
  return {
    available: false,
    response: apiErrorResponse({
      status: 400,
      errorCode: ApiErrorCode.ORGANIZATION_CONTEXT_INVALID,
      message: 'Organization context is invalid',
      headers: privateSettingHeaders(true),
    }),
  };
}

function unauthenticated(headers: Headers): NextResponse {
  return apiErrorResponse({
    status: 401,
    errorCode: ApiErrorCode.AUTH_UNAUTHORIZED,
    message: 'Authentication required',
    headers,
  });
}

function forbidden(headers: Headers): NextResponse {
  return apiErrorResponse({
    status: 403,
    errorCode: ApiErrorCode.PERMISSION_DENIED,
    message: 'Operation forbidden',
    headers,
  });
}

function unavailable(message: string): SettingRouteResolution {
  return {
    available: false,
    response: apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message,
      headers: publicNoStoreHeaders(),
    }),
  };
}
