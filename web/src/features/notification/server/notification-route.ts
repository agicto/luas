import 'server-only';

import type { NextResponse } from 'next/server';

import {
  apiErrorResponse,
  apiJsonBodyErrorResponse,
  apiValidationErrorResponse,
} from '@/app/api/_shared/error-response';
import { readJsonBody } from '@/app/api/_shared/json-body';
import { guardSameOriginMutation } from '@/app/api/_shared/mock-bff';
import {
  apiPaginatedResponse,
  apiSuccessResponse,
} from '@/app/api/_shared/success-response';
import { env } from '@/config/env';
import { isWebFeatureEnabled } from '@/config/features';
import { isMockBffEnabled } from '@/config/mock-bff';
import { serverEnv } from '@/config/server-env';
import { getApiSessionToken } from '@/features/auth/server/api-session';
import { getSessionUser } from '@/features/auth/server/session';
import type { AuthUser } from '@/features/auth/types';
import {
  markNotificationsReadSchema,
  notificationPreferenceSchema,
  notificationRouteIdSchema,
  replaceNotificationReadStateSchema,
} from '@/features/notification/schemas';
import { mockNotificationStore } from '@/features/notification/server/mock-notification-store';
import type { NotificationFilter } from '@/features/notification/types';
import { ApiErrorCode } from '@/http/codes';
import { forwardAuthenticatedGoApi } from '@/server/api-adapter/authenticated-route';
import { privateNoStoreResponse } from '@/server/http/private-response';

type NotificationBackend = 'go-api' | 'mock';

interface NotificationRouteEnvironment {
  adapterEnabled: boolean;
  mockBffEnabled: boolean;
  nodeEnv: 'development' | 'production' | 'test';
}

type NotificationRouteResolution =
  | { available: true; backend: NotificationBackend }
  | { available: false; response: NextResponse };

type AuthenticatedNotificationBackend =
  | { authenticated: true; backend: 'go-api'; accessToken: string }
  | { authenticated: true; backend: 'mock'; user: AuthUser }
  | { authenticated: false; response: NextResponse };

const maxMutationBodyBytes = 4 * 1_024;

export function resolveNotificationRoute(
  environment: NotificationRouteEnvironment = currentEnvironment()
): NotificationRouteResolution {
  if (!isWebFeatureEnabled('notification')) {
    return unavailable('Notification Web feature is disabled');
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
  return unavailable('Notification backend is unavailable');
}

export function privateNotificationResponse<T extends Response>(response: T): T {
  return privateNoStoreResponse(response, ['Cookie']);
}

export async function listNotificationsRoute(request: Request): Promise<NextResponse> {
  const route = await resolveAuthenticatedRoute();
  if (!route.available) return route.response;
  const rawStatus = new URL(request.url).searchParams.get('status') ?? 'all';
  if (rawStatus !== 'all' && rawStatus !== 'unread') {
    return apiErrorResponse({
      status: 422,
      errorCode: ApiErrorCode.COMMON_VALIDATION_FAILED,
      message: 'Invalid notification filter',
    });
  }
  const pagination = paginationFromRequest(request);

  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'notifications',
      accessToken: route.authentication.accessToken,
      searchParams: pagination.searchParams,
    });
  }
  const result = mockNotificationStore.list(
    route.authentication.user,
    pagination.page,
    pagination.perPage,
    pagination.status
  );
  return apiPaginatedResponse(result.items, result.meta, result.links);
}

export async function getNotificationStatusRoute(request: Request): Promise<NextResponse> {
  const route = await resolveAuthenticatedRoute();
  if (!route.available) return route.response;
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'notification-status',
      accessToken: route.authentication.accessToken,
    });
  }
  return apiSuccessResponse(mockNotificationStore.status(route.authentication.user));
}

export async function replaceNotificationReadStateRoute(
  request: Request,
  rawNotificationId: string
): Promise<NextResponse> {
  const route = await resolveMutationRoute(request);
  if (!route.available) return route.response;
  const notificationId = parseRouteId(rawNotificationId);
  if (!notificationId) return notificationNotFound();
  const payload = await readJsonBody(request, maxMutationBodyBytes);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = replaceNotificationReadStateSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid notification read state', parsed.error);
  }
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'PATCH',
      path: `notifications/${notificationId}`,
      accessToken: route.authentication.accessToken,
      body: parsed.data,
      fieldMap: { is_read: 'is_read' },
    });
  }
  const item = mockNotificationStore.replaceReadState(
    route.authentication.user,
    notificationId,
    parsed.data.is_read
  );
  return item ? apiSuccessResponse(item) : notificationNotFound();
}

export async function markNotificationsReadRoute(request: Request): Promise<NextResponse> {
  const route = await resolveMutationRoute(request);
  if (!route.available) return route.response;
  const payload = await readJsonBody(request, maxMutationBodyBytes);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = markNotificationsReadSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid notification read high-water mark', parsed.error);
  }
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'PUT',
      path: 'notification-read-state',
      accessToken: route.authentication.accessToken,
      body: parsed.data,
      fieldMap: { through_id: 'through_id' },
    });
  }
  return apiSuccessResponse(
    mockNotificationStore.markReadThrough(route.authentication.user, parsed.data.through_id)
  );
}

export async function getNotificationPreferenceRoute(request: Request): Promise<NextResponse> {
  const route = await resolveAuthenticatedRoute();
  if (!route.available) return route.response;
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'notification-preferences',
      accessToken: route.authentication.accessToken,
    });
  }
  return apiSuccessResponse(mockNotificationStore.preference(route.authentication.user));
}

export async function replaceNotificationPreferenceRoute(request: Request): Promise<NextResponse> {
  const route = await resolveMutationRoute(request);
  if (!route.available) return route.response;
  const payload = await readJsonBody(request, maxMutationBodyBytes);
  if (!payload.ok) return apiJsonBodyErrorResponse(payload.error);
  const parsed = notificationPreferenceSchema.safeParse(payload.data);
  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid notification preferences', parsed.error);
  }
  if (route.authentication.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'PUT',
      path: 'notification-preferences',
      accessToken: route.authentication.accessToken,
      body: parsed.data,
      fieldMap: {
        in_app_enabled: 'in_app_enabled',
        email_enabled: 'email_enabled',
      },
    });
  }
  return apiSuccessResponse(
    mockNotificationStore.replacePreference(route.authentication.user, parsed.data)
  );
}

async function resolveMutationRoute(request: Request) {
  const resolution = resolveNotificationRoute();
  if (!resolution.available) return resolution;
  const guard = guardSameOriginMutation(request);
  return guard
    ? { available: false as const, response: guard }
    : resolveAuthenticatedRoute(resolution);
}

async function resolveAuthenticatedRoute(
  resolution: NotificationRouteResolution = resolveNotificationRoute()
) {
  if (!resolution.available) return resolution;
  const authentication = await authenticateNotificationBackend(resolution.backend);
  return authentication.authenticated
    ? { available: true as const, authentication }
    : { available: false as const, response: authentication.response };
}

async function authenticateNotificationBackend(
  backend: NotificationBackend
): Promise<AuthenticatedNotificationBackend> {
  if (backend === 'go-api') {
    const accessToken = await getApiSessionToken();
    return accessToken
      ? { authenticated: true, backend, accessToken }
      : { authenticated: false, response: unauthenticated() };
  }
  const user = await getSessionUser();
  return user
    ? { authenticated: true, backend, user }
    : { authenticated: false, response: unauthenticated() };
}

function paginationFromRequest(request: Request) {
  const input = new URL(request.url).searchParams;
  const page = positiveInteger(input.get('page'), 1);
  const perPage = Math.min(positiveInteger(input.get('per_page'), 10), 100);
  const status = (input.get('status') ?? 'all') as NotificationFilter;
  return {
    page,
    perPage,
    status,
    searchParams: new URLSearchParams({
      page: String(page),
      per_page: String(perPage),
      status,
    }),
  };
}

function positiveInteger(value: string | null, fallback: number): number {
  if (!value || !/^[1-9]\d*$/.test(value)) return fallback;
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) ? parsed : fallback;
}

function parseRouteId(value: string): number | null {
  const parsed = notificationRouteIdSchema.safeParse(value);
  if (!parsed.success) return null;
  const id = Number(parsed.data);
  return Number.isSafeInteger(id) && id > 0 ? id : null;
}

function currentEnvironment(): NotificationRouteEnvironment {
  return {
    adapterEnabled: serverEnv.API_ADAPTER_ENABLED,
    mockBffEnabled: serverEnv.MOCK_BFF_ENABLED,
    nodeEnv: env.NODE_ENV,
  };
}

function unauthenticated(): NextResponse {
  return apiErrorResponse({
    status: 401,
    errorCode: ApiErrorCode.AUTH_UNAUTHORIZED,
    message: 'Authentication required',
  });
}

function unavailable(message: string): NotificationRouteResolution {
  return {
    available: false,
    response: apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message,
    }),
  };
}

function notificationNotFound(): NextResponse {
  return apiErrorResponse({
    status: 404,
    errorCode: ApiErrorCode.NOTIFICATION_NOT_FOUND,
    message: 'Notification not found',
  });
}
