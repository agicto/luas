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
  apiNoContentResponse,
  apiPaginatedResponse,
  apiSuccessResponse,
} from '@/app/api/_shared/success-response';
import { env } from '@/config/env';
import { isWebFeatureEnabled } from '@/config/features';
import { isMockBffEnabled } from '@/config/mock-bff';
import { serverEnv } from '@/config/server-env';
import {
  authenticateOrganizationBackend,
  parseOrganizationId,
  paginationFromRequest,
} from '@/features/organization/server/organization-route';
import type { OrganizationContext } from '@/features/organization/types';
import {
  webhookEndpointInputSchema,
  webhookEndpointStatusInputSchema,
  webhookIdempotencyKeySchema,
  webhookRouteIdSchema,
} from '@/features/webhook/schemas';
import {
  mockWebhookStore,
  type MockWebhookStoreError,
} from '@/features/webhook/server/mock-webhook-store';
import type { WebhookDelivery } from '@/features/webhook/types';
import { mockOrganizationStore } from '@/features/organization/server/mock-organization-store';
import { ApiErrorCode } from '@/http/codes';
import { forwardAuthenticatedGoApi } from '@/server/api-adapter/authenticated-route';
import { privateNoStoreHeaders, privateNoStoreResponse } from '@/server/http/private-response';

type WebhookBackend = 'go-api' | 'mock';

interface WebhookRouteEnvironment {
  adapterEnabled: boolean;
  mockBffEnabled: boolean;
  nodeEnv: 'development' | 'production' | 'test';
  organizationFeatureEnabled: boolean;
  webhookFeatureEnabled: boolean;
}

type WebhookRouteResolution =
  { available: true; backend: WebhookBackend } | { available: false; response: NextResponse };

type WebhookManagerTarget =
  | {
      available: true;
      organizationId: number;
      authentication: Awaited<ReturnType<typeof authenticateOrganizationBackend>> & {
        authenticated: true;
      };
      context?: OrganizationContext;
    }
  | { available: false; response: NextResponse };

const endpointETagPattern = /^"webhook-endpoint-v([1-9]\d{0,19})"$/u;
const deliveryStatuses = new Set<WebhookDelivery['status']>([
  'pending',
  'processing',
  'delivered',
  'failed',
  'canceled',
]);
const maxManagementBodyBytes = 16 * 1_024;

export function resolveWebhookRoute(
  environment: WebhookRouteEnvironment = currentEnvironment()
): WebhookRouteResolution {
  if (!environment.organizationFeatureEnabled || !environment.webhookFeatureEnabled) {
    return unavailable('Webhook Web feature is disabled');
  }
  if (environment.adapterEnabled) return { available: true, backend: 'go-api' };
  if (isMockBffEnabled({ enabled: environment.mockBffEnabled, nodeEnv: environment.nodeEnv })) {
    return { available: true, backend: 'mock' };
  }
  return unavailable('Webhook backend is unavailable');
}

export function privateWebhookResponse<T extends Response>(response: T): T {
  privateNoStoreResponse(response, ['Cookie', 'Organization-Id']);
  response.headers.set('pragma', 'no-cache');
  return response;
}

export async function webhookEventTypesRoute(request: Request): Promise<NextResponse> {
  const target = await resolveWebhookManager(request);
  if (!target.available) return privateWebhookResponse(target.response);
  if (target.authentication.backend === 'go-api') {
    return privateWebhookResponse(
      await forwardAuthenticatedGoApi(request, {
        method: 'GET',
        path: 'webhook-event-types',
        accessToken: target.authentication.accessToken,
        organizationId: String(target.organizationId),
      })
    );
  }
  return apiSuccessResponse(mockWebhookStore.eventTypes(), { headers: webhookHeaders() });
}

export async function listWebhookEndpointsRoute(request: Request): Promise<NextResponse> {
  const target = await resolveWebhookManager(request);
  if (!target.available) return privateWebhookResponse(target.response);
  const pagination = paginationFromRequest(request);
  if (target.authentication.backend === 'go-api') {
    return privateWebhookResponse(
      await forwardAuthenticatedGoApi(request, {
        method: 'GET',
        path: 'webhook-endpoints',
        accessToken: target.authentication.accessToken,
        organizationId: String(target.organizationId),
        searchParams: pagination.searchParams,
      })
    );
  }
  const result = mockWebhookStore.listEndpoints(target.organizationId, pagination);
  return apiPaginatedResponse(result.items, result.meta, result.links, webhookHeaders());
}

export async function createWebhookEndpointRoute(request: Request): Promise<NextResponse> {
  const resolution = resolveWebhookRoute();
  if (!resolution.available) return privateWebhookResponse(resolution.response);
  const originGuard = guardSameOriginMutation(request);
  if (originGuard) return privateWebhookResponse(originGuard);
  const target = await resolveWebhookManager(request, resolution);
  if (!target.available) return privateWebhookResponse(target.response);
  const input = await endpointInput(request);
  if (!input.ok) return privateWebhookResponse(input.response);

  if (target.authentication.backend === 'go-api') {
    return privateWebhookResponse(
      await forwardAuthenticatedGoApi(request, {
        method: 'POST',
        path: 'webhook-endpoints',
        accessToken: target.authentication.accessToken,
        organizationId: String(target.organizationId),
        body: input.value,
        fieldMap: { name: 'name', url: 'url', event_types: 'event_types' },
      })
    );
  }
  const result = mockWebhookStore.createEndpoint(target.organizationId, input.value);
  const headers = webhookHeaders(endpointETag(result.endpoint.version));
  return apiSuccessResponse(result, { status: 201, message: 'created', headers });
}

export async function updateWebhookEndpointRoute(
  request: Request,
  rawEndpointId: string
): Promise<NextResponse> {
  const mutation = await resolveEndpointMutation(request, rawEndpointId);
  if (!mutation.available) return privateWebhookResponse(mutation.response);
  const input = await endpointInput(request);
  if (!input.ok) return privateWebhookResponse(input.response);
  const { target, endpointId, etag, version } = mutation;

  if (target.authentication.backend === 'go-api') {
    return privateWebhookResponse(
      await forwardAuthenticatedGoApi(request, {
        method: 'PATCH',
        path: `webhook-endpoints/${endpointId}`,
        accessToken: target.authentication.accessToken,
        organizationId: String(target.organizationId),
        ifMatch: etag,
        body: input.value,
        fieldMap: { name: 'name', url: 'url', event_types: 'event_types' },
      })
    );
  }
  return endpointMutationResponse(
    mockWebhookStore.updateEndpoint(target.organizationId, endpointId, version, input.value)
  );
}

export async function deleteWebhookEndpointRoute(
  request: Request,
  rawEndpointId: string
): Promise<NextResponse> {
  const mutation = await resolveEndpointMutation(request, rawEndpointId);
  if (!mutation.available) return privateWebhookResponse(mutation.response);
  const { target, endpointId, etag, version } = mutation;

  if (target.authentication.backend === 'go-api') {
    return privateWebhookResponse(
      await forwardAuthenticatedGoApi(request, {
        method: 'DELETE',
        path: `webhook-endpoints/${endpointId}`,
        accessToken: target.authentication.accessToken,
        organizationId: String(target.organizationId),
        ifMatch: etag,
      })
    );
  }
  const result = mockWebhookStore.deleteEndpoint(target.organizationId, endpointId, version);
  if (result !== true) return webhookStoreError(result);
  return apiNoContentResponse(webhookHeaders());
}

export async function replaceWebhookEndpointStatusRoute(
  request: Request,
  rawEndpointId: string
): Promise<NextResponse> {
  const mutation = await resolveEndpointMutation(request, rawEndpointId);
  if (!mutation.available) return privateWebhookResponse(mutation.response);
  const payload = await readJsonBody(request, maxManagementBodyBytes);
  if (!payload.ok) return privateWebhookResponse(apiJsonBodyErrorResponse(payload.error));
  const input = webhookEndpointStatusInputSchema.safeParse(payload.data);
  if (!input.success) {
    return privateWebhookResponse(
      apiValidationErrorResponse('Invalid webhook endpoint status', input.error)
    );
  }
  const { target, endpointId, etag, version } = mutation;

  if (target.authentication.backend === 'go-api') {
    return privateWebhookResponse(
      await forwardAuthenticatedGoApi(request, {
        method: 'PUT',
        path: `webhook-endpoints/${endpointId}/status`,
        accessToken: target.authentication.accessToken,
        organizationId: String(target.organizationId),
        ifMatch: etag,
        body: input.data,
        fieldMap: { enabled: 'enabled' },
      })
    );
  }
  return endpointMutationResponse(
    mockWebhookStore.replaceEndpointStatus(
      target.organizationId,
      endpointId,
      version,
      input.data.enabled
    )
  );
}

export async function rotateWebhookEndpointSecretRoute(
  request: Request,
  rawEndpointId: string
): Promise<NextResponse> {
  const mutation = await resolveEndpointMutation(request, rawEndpointId);
  if (!mutation.available) return privateWebhookResponse(mutation.response);
  const { target, endpointId, etag, version } = mutation;

  if (target.authentication.backend === 'go-api') {
    return privateWebhookResponse(
      await forwardAuthenticatedGoApi(request, {
        method: 'POST',
        path: `webhook-endpoints/${endpointId}/secret-rotations`,
        accessToken: target.authentication.accessToken,
        organizationId: String(target.organizationId),
        ifMatch: etag,
      })
    );
  }
  const result = mockWebhookStore.rotateEndpointSecret(target.organizationId, endpointId, version);
  if (typeof result === 'string') return webhookStoreError(result);
  const headers = webhookHeaders(endpointETag(result.endpoint.version));
  return apiSuccessResponse(result, { status: 201, message: 'created', headers });
}

export async function testWebhookEndpointRoute(
  request: Request,
  rawEndpointId: string
): Promise<NextResponse> {
  const resolution = resolveWebhookRoute();
  if (!resolution.available) return privateWebhookResponse(resolution.response);
  const originGuard = guardSameOriginMutation(request);
  if (originGuard) return privateWebhookResponse(originGuard);
  const target = await resolveWebhookManager(request, resolution);
  if (!target.available) return privateWebhookResponse(target.response);
  const endpointId = parseWebhookRouteId(rawEndpointId);
  if (!endpointId) return webhookEndpointNotFound();
  const idempotencyKey = parseIdempotencyKey(request);
  if (!idempotencyKey.ok) return idempotencyKey.response;

  if (target.authentication.backend === 'go-api') {
    return privateWebhookResponse(
      await forwardAuthenticatedGoApi(request, {
        method: 'POST',
        path: `webhook-endpoints/${endpointId}/tests`,
        accessToken: target.authentication.accessToken,
        organizationId: String(target.organizationId),
        idempotencyKey: idempotencyKey.value,
      })
    );
  }
  const result = mockWebhookStore.testEndpoint(
    target.organizationId,
    endpointId,
    idempotencyKey.value
  );
  if (typeof result === 'string') return webhookStoreError(result);
  return apiSuccessResponse(result, {
    status: 202,
    message: 'accepted',
    headers: webhookHeaders(),
  });
}

export async function listWebhookDeliveriesRoute(request: Request): Promise<NextResponse> {
  const target = await resolveWebhookManager(request);
  if (!target.available) return privateWebhookResponse(target.response);
  const filters = deliveryFilters(request);
  if (!filters.ok) return filters.response;

  if (target.authentication.backend === 'go-api') {
    return privateWebhookResponse(
      await forwardAuthenticatedGoApi(request, {
        method: 'GET',
        path: 'webhook-deliveries',
        accessToken: target.authentication.accessToken,
        organizationId: String(target.organizationId),
        searchParams: filters.searchParams,
      })
    );
  }
  const result = mockWebhookStore.listDeliveries(target.organizationId, filters);
  return apiPaginatedResponse(result.items, result.meta, result.links, webhookHeaders());
}

export async function listWebhookAttemptsRoute(
  request: Request,
  rawDeliveryId: string
): Promise<NextResponse> {
  const target = await resolveWebhookManager(request);
  if (!target.available) return privateWebhookResponse(target.response);
  const deliveryId = parseWebhookRouteId(rawDeliveryId);
  if (!deliveryId) return webhookDeliveryNotFound();
  const pagination = paginationFromRequest(request);

  if (target.authentication.backend === 'go-api') {
    return privateWebhookResponse(
      await forwardAuthenticatedGoApi(request, {
        method: 'GET',
        path: `webhook-deliveries/${deliveryId}/attempts`,
        accessToken: target.authentication.accessToken,
        organizationId: String(target.organizationId),
        searchParams: pagination.searchParams,
      })
    );
  }
  const result = mockWebhookStore.listAttempts(target.organizationId, deliveryId, pagination);
  if (typeof result === 'string') return webhookStoreError(result);
  return apiPaginatedResponse(result.items, result.meta, result.links, webhookHeaders());
}

async function resolveWebhookManager(
  request: Request,
  resolution: WebhookRouteResolution = resolveWebhookRoute()
): Promise<WebhookManagerTarget> {
  if (!resolution.available) return resolution;
  const authentication = await authenticateOrganizationBackend(resolution.backend);
  if (!authentication.authenticated) {
    return { available: false, response: authentication.response };
  }
  const rawOrganizationId = request.headers.get('organization-id');
  if (rawOrganizationId === null) return organizationContextRequired();
  const organizationId = parseOrganizationId(rawOrganizationId);
  if (!organizationId) return organizationContextInvalid();
  if (authentication.backend === 'go-api') {
    return { available: true, organizationId, authentication };
  }
  const context = mockOrganizationStore.resolveContext(authentication.user, organizationId);
  if (!context) return organizationNotFound();
  if (context.role !== 'owner' && context.role !== 'admin') return forbidden();
  return { available: true, organizationId, authentication, context };
}

async function resolveEndpointMutation(request: Request, rawEndpointId: string) {
  const resolution = resolveWebhookRoute();
  if (!resolution.available) return { available: false as const, response: resolution.response };
  const originGuard = guardSameOriginMutation(request);
  if (originGuard) return { available: false as const, response: originGuard };
  const target = await resolveWebhookManager(request, resolution);
  if (!target.available) return target;
  const endpointId = parseWebhookRouteId(rawEndpointId);
  if (!endpointId) return { available: false as const, response: webhookEndpointNotFound() };
  const expected = expectedEndpointVersion(request);
  if (!expected.ok) return { available: false as const, response: expected.response };
  return {
    available: true as const,
    target,
    endpointId,
    etag: expected.etag,
    version: expected.version,
  };
}

async function endpointInput(request: Request) {
  const payload = await readJsonBody(request, maxManagementBodyBytes);
  if (!payload.ok) {
    return { ok: false as const, response: apiJsonBodyErrorResponse(payload.error) };
  }
  const input = webhookEndpointInputSchema.safeParse(payload.data);
  if (input.success) return { ok: true as const, value: input.data };
  const issueFields = new Set(input.error.issues.map(issue => String(issue.path[0] ?? 'body')));
  if (issueFields.has('url')) {
    return {
      ok: false as const,
      response: apiErrorResponse({
        status: 422,
        errorCode: ApiErrorCode.WEBHOOK_INVALID_TARGET,
        message: 'Webhook target is invalid',
        headers: webhookHeaders(),
      }),
    };
  }
  if (issueFields.has('event_types')) {
    return {
      ok: false as const,
      response: apiErrorResponse({
        status: 422,
        errorCode: ApiErrorCode.WEBHOOK_INVALID_EVENT_TYPE,
        message: 'Webhook event type is invalid',
        headers: webhookHeaders(),
      }),
    };
  }
  return {
    ok: false as const,
    response: apiValidationErrorResponse('Invalid webhook endpoint', input.error),
  };
}

function expectedEndpointVersion(request: Request) {
  const etag = request.headers.get('if-match');
  if (etag === null) {
    return {
      ok: false as const,
      response: apiErrorResponse({
        status: 428,
        errorCode: ApiErrorCode.WEBHOOK_PRECONDITION_REQUIRED,
        message: 'Webhook endpoint version is required',
        headers: webhookHeaders(),
      }),
    };
  }
  const match = endpointETagPattern.exec(etag);
  const version = match ? Number(match[1]) : Number.NaN;
  if (!match || !Number.isSafeInteger(version)) {
    return {
      ok: false as const,
      response: invalidInput('Invalid If-Match header'),
    };
  }
  return { ok: true as const, etag, version };
}

function parseIdempotencyKey(request: Request) {
  const value = request.headers.get('idempotency-key');
  if (value === null || !webhookIdempotencyKeySchema.safeParse(value).success) {
    return {
      ok: false as const,
      response: invalidInput('Valid Idempotency-Key header required'),
    };
  }
  return { ok: true as const, value };
}

function deliveryFilters(request: Request) {
  const pagination = paginationFromRequest(request);
  const input = new URL(request.url).searchParams;
  const rawEndpointId = input.get('endpoint_id');
  let endpointId: number | undefined;
  if (rawEndpointId !== null) {
    const parsedEndpointId = parseWebhookRouteId(rawEndpointId);
    if (parsedEndpointId === null) {
      return { ok: false as const, response: invalidInput('Invalid endpoint_id filter') };
    }
    endpointId = parsedEndpointId;
  }
  const rawStatus = input.get('status');
  if (rawStatus !== null && !deliveryStatuses.has(rawStatus as WebhookDelivery['status'])) {
    return { ok: false as const, response: invalidInput('Invalid status filter') };
  }
  const searchParams = pagination.searchParams;
  if (endpointId !== undefined) searchParams.set('endpoint_id', String(endpointId));
  if (rawStatus !== null) searchParams.set('status', rawStatus);
  return {
    ok: true as const,
    page: pagination.page,
    perPage: pagination.perPage,
    searchParams,
    ...(endpointId === undefined ? {} : { endpointId }),
    ...(rawStatus === null ? {} : { status: rawStatus as WebhookDelivery['status'] }),
  };
}

function parseWebhookRouteId(value: string): number | null {
  const parsed = webhookRouteIdSchema.safeParse(value);
  if (!parsed.success) return null;
  const id = Number(parsed.data);
  return Number.isSafeInteger(id) ? id : null;
}

function endpointMutationResponse(
  result: ReturnType<
    typeof mockWebhookStore.updateEndpoint | typeof mockWebhookStore.replaceEndpointStatus
  >
): NextResponse {
  if (typeof result === 'string') return webhookStoreError(result);
  return apiSuccessResponse(result, { headers: webhookHeaders(endpointETag(result.version)) });
}

function webhookStoreError(error: MockWebhookStoreError): NextResponse {
  if (error === 'endpoint_not_found') return webhookEndpointNotFound();
  if (error === 'delivery_not_found') return webhookDeliveryNotFound();
  if (error === 'version_conflict') {
    return apiErrorResponse({
      status: 409,
      errorCode: ApiErrorCode.WEBHOOK_ENDPOINT_VERSION_CONFLICT,
      message: 'Webhook endpoint version conflict',
      headers: webhookHeaders(),
    });
  }
  return apiErrorResponse({
    status: 409,
    errorCode: ApiErrorCode.WEBHOOK_REPLAY_NOT_ALLOWED,
    message: 'Webhook delivery is not allowed',
    headers: webhookHeaders(),
  });
}

function webhookEndpointNotFound(): NextResponse {
  return apiErrorResponse({
    status: 404,
    errorCode: ApiErrorCode.WEBHOOK_ENDPOINT_NOT_FOUND,
    message: 'Webhook endpoint not found',
    headers: webhookHeaders(),
  });
}

function webhookDeliveryNotFound(): NextResponse {
  return apiErrorResponse({
    status: 404,
    errorCode: ApiErrorCode.WEBHOOK_DELIVERY_NOT_FOUND,
    message: 'Webhook delivery not found',
    headers: webhookHeaders(),
  });
}

function organizationContextRequired(): WebhookManagerTarget {
  return {
    available: false,
    response: apiErrorResponse({
      status: 400,
      errorCode: ApiErrorCode.ORGANIZATION_CONTEXT_REQUIRED,
      message: 'Organization context is required',
      headers: webhookHeaders(),
    }),
  };
}

function organizationContextInvalid(): WebhookManagerTarget {
  return {
    available: false,
    response: apiErrorResponse({
      status: 400,
      errorCode: ApiErrorCode.ORGANIZATION_CONTEXT_INVALID,
      message: 'Organization context is invalid',
      headers: webhookHeaders(),
    }),
  };
}

function organizationNotFound(): WebhookManagerTarget {
  return {
    available: false,
    response: apiErrorResponse({
      status: 404,
      errorCode: ApiErrorCode.ORGANIZATION_NOT_FOUND,
      message: 'Organization not found',
      headers: webhookHeaders(),
    }),
  };
}

function forbidden(): WebhookManagerTarget {
  return {
    available: false,
    response: apiErrorResponse({
      status: 403,
      errorCode: ApiErrorCode.PERMISSION_DENIED,
      message: 'Webhook management forbidden',
      headers: webhookHeaders(),
    }),
  };
}

function invalidInput(message: string): NextResponse {
  return apiErrorResponse({
    status: 400,
    errorCode: ApiErrorCode.COMMON_INVALID_INPUT,
    message,
    headers: webhookHeaders(),
  });
}

function unavailable(message: string): WebhookRouteResolution {
  return {
    available: false,
    response: apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message,
      headers: { 'cache-control': 'no-store', pragma: 'no-cache' },
    }),
  };
}

function currentEnvironment(): WebhookRouteEnvironment {
  return {
    adapterEnabled: serverEnv.API_ADAPTER_ENABLED,
    mockBffEnabled: serverEnv.MOCK_BFF_ENABLED,
    nodeEnv: env.NODE_ENV,
    organizationFeatureEnabled: isWebFeatureEnabled('organization'),
    webhookFeatureEnabled: isWebFeatureEnabled('webhook'),
  };
}

function webhookHeaders(etag?: string): Headers {
  const headers = privateNoStoreHeaders(undefined, ['Cookie', 'Organization-Id']);
  headers.set('pragma', 'no-cache');
  if (etag) headers.set('etag', etag);
  return headers;
}

function endpointETag(version: number): string {
  return `"webhook-endpoint-v${version}"`;
}
