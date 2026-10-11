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
import {
  KNOWLEDGE_BODY_MAX_LENGTH,
  KNOWLEDGE_PAGE_MAX_SIZE,
  KNOWLEDGE_QUERY_MAX_LENGTH,
  KNOWLEDGE_TITLE_MAX_LENGTH,
  characterCount,
  knowledgeCursorSchema,
  knowledgeRouteIdSchema,
} from '@/features/knowledge/schemas';
import {
  decodeKnowledgeCursor,
  mockKnowledgeStore,
  type MockKnowledgeDeleteResult,
  type MockKnowledgeWriteResult,
} from '@/features/knowledge/server/mock-knowledge-store';
import type { KnowledgeArticle, KnowledgeArticlePage } from '@/features/knowledge/types';
import { mockOrganizationStore } from '@/features/organization/server/mock-organization-store';
import { parseOrganizationId } from '@/features/organization/server/organization-route';
import type { OrganizationContext } from '@/features/organization/types';
import { ApiErrorCode } from '@/http/codes';
import { forwardAuthenticatedGoApi } from '@/server/api-adapter/authenticated-route';
import { privateNoStoreHeaders, privateNoStoreResponse } from '@/server/http/private-response';

/**
 * Browser route boundary of the knowledge feature. Every handler selects the Go API adapter or
 * the development mock, requires the explicit `organization,knowledge` feature selection, and
 * resolves the active organization from `Organization-Id`. Writes check availability, then the
 * browser origin, then authentication, before reading a body. Organization membership and the
 * owner/admin write rule stay authoritative in the API; the mock reproduces them for development.
 */

type KnowledgeBackend = 'go-api' | 'mock';

interface KnowledgeRouteEnvironment {
  adapterEnabled: boolean;
  featureEnabled: boolean;
  mockBffEnabled: boolean;
  nodeEnv: 'development' | 'production' | 'test';
}

type KnowledgeRouteResolution =
  { available: true; backend: KnowledgeBackend } | { available: false; response: NextResponse };

type AuthenticatedKnowledgeBackend =
  | { authenticated: true; backend: 'go-api'; accessToken: string }
  | { authenticated: true; backend: 'mock'; user: AuthUser }
  | { authenticated: false; response: NextResponse };

type Authentication = Extract<AuthenticatedKnowledgeBackend, { authenticated: true }>;

type KnowledgeTarget =
  | { available: true; backend: 'go-api'; organizationId: number; accessToken: string }
  | { available: true; backend: 'mock'; organizationId: number; context: OrganizationContext }
  | { available: false; response: NextResponse };

type Parsed<T> = { ok: true; value: T } | { ok: false; response: NextResponse };

const articleVersionPattern = /^"knowledge-article-v([1-9]\d{0,15})"$/u;
const defaultPerPage = 15;
// A full body is 200,000 characters; JSON escaping and multi-byte UTF-8 need headroom.
const maxArticleBodyBytes = 1_024 * 1_024;

export function resolveKnowledgeRoute(
  environment: KnowledgeRouteEnvironment = currentEnvironment()
): KnowledgeRouteResolution {
  if (!environment.featureEnabled) return unavailable('Knowledge Web feature is disabled');
  if (environment.adapterEnabled) return { available: true, backend: 'go-api' };
  if (isMockBffEnabled({ enabled: environment.mockBffEnabled, nodeEnv: environment.nodeEnv })) {
    return { available: true, backend: 'mock' };
  }
  return unavailable('Knowledge backend is unavailable');
}

export function privateKnowledgeResponse<T extends Response>(response: T): T {
  return privateNoStoreResponse(response, ['Cookie', 'Organization-Id']);
}

export async function listKnowledgeArticlesRoute(request: Request): Promise<NextResponse> {
  const target = await resolveKnowledgeReadRoute(request);
  if (!target.available) return target.response;
  const query = new URL(request.url).searchParams;
  const cursors = query.getAll('cursor');
  if (cursors.length > 1 || query.has('page')) return invalidInput('Invalid cursor');
  const cursor = cursors[0] ?? '';
  if (cursor !== '' && !knowledgeCursorSchema.safeParse(cursor).success) {
    return invalidInput('Invalid cursor');
  }
  const perPage = parsePerPage(query.get('per_page'));

  if (target.backend === 'go-api') {
    const searchParams = new URLSearchParams({ per_page: String(perPage) });
    if (cursor !== '') searchParams.set('cursor', cursor);
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'knowledge-articles',
      accessToken: target.accessToken,
      organizationId: String(target.organizationId),
      searchParams,
    });
  }

  const afterId = cursor === '' ? null : decodeKnowledgeCursor(cursor);
  if (cursor !== '' && afterId === null) return invalidInput('Invalid cursor');
  return knowledgePageResponse(mockKnowledgeStore.list(target.organizationId, afterId, perPage));
}

export async function getKnowledgeArticleRoute(
  request: Request,
  rawId: string
): Promise<NextResponse> {
  const target = await resolveKnowledgeReadRoute(request);
  if (!target.available) return target.response;
  const articleId = parseArticleId(rawId);
  if (!articleId.ok) return articleId.response;

  if (target.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: `knowledge-articles/${articleId.value}`,
      accessToken: target.accessToken,
      organizationId: String(target.organizationId),
    });
  }
  const article = mockKnowledgeStore.get(target.organizationId, articleId.value);
  return article ? articleResponse(article) : articleNotFound();
}

export async function searchKnowledgeArticlesRoute(request: Request): Promise<NextResponse> {
  const target = await resolveKnowledgeReadRoute(request);
  if (!target.available) return target.response;
  const values = new URL(request.url).searchParams.getAll('q');
  const query = values[0];
  if (values.length !== 1 || query === undefined || query === '') {
    return validationFailed({ q: ['The q field is required'] });
  }
  if (characterCount(query) > KNOWLEDGE_QUERY_MAX_LENGTH) {
    return validationFailed({ q: ['The q field is too long'] });
  }
  if (query.trim() === '') return invalidInput('Search query is blank');

  if (target.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'GET',
      path: 'knowledge-search',
      accessToken: target.accessToken,
      organizationId: String(target.organizationId),
      searchParams: new URLSearchParams({ q: query }),
    });
  }
  return apiSuccessResponse(mockKnowledgeStore.search(target.organizationId, query), {
    headers: privateKnowledgeHeaders(),
  });
}

export async function createKnowledgeArticleRoute(request: Request): Promise<NextResponse> {
  const target = await resolveKnowledgeMutationRoute(request);
  if (!target.available) return target.response;
  const manager = requireManager(target);
  if (manager) return manager;
  const input = await articleBody(request, 'create');
  if (!input.ok) return input.response;

  if (target.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'POST',
      path: 'knowledge-articles',
      accessToken: target.accessToken,
      organizationId: String(target.organizationId),
      body: input.value,
      fieldMap: { title: 'title', body: 'body' },
    });
  }
  const title = input.value.title?.trim() ?? '';
  if (title === '') return invalidArticle();
  return mockWriteResponse(
    mockKnowledgeStore.create(target.organizationId, target.context.user_id, {
      title,
      body: input.value.body ?? '',
    }),
    201
  );
}

export async function updateKnowledgeArticleRoute(
  request: Request,
  rawId: string
): Promise<NextResponse> {
  const target = await resolveKnowledgeMutationRoute(request);
  if (!target.available) return target.response;
  const manager = requireManager(target);
  if (manager) return manager;
  const articleId = parseArticleId(rawId);
  if (!articleId.ok) return articleId.response;
  const expectedVersion = expectedArticleVersion(request);
  if (!expectedVersion.ok) return expectedVersion.response;
  const input = await articleBody(request, 'update');
  if (!input.ok) return input.response;

  if (target.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'PATCH',
      path: `knowledge-articles/${articleId.value}`,
      accessToken: target.accessToken,
      organizationId: String(target.organizationId),
      ifMatch: expectedVersion.value.etag,
      body: input.value,
      fieldMap: { title: 'title', body: 'body' },
    });
  }
  const title = input.value.title?.trim();
  if ((title === undefined && input.value.body === undefined) || title === '') {
    return invalidArticle();
  }
  return mockWriteResponse(
    mockKnowledgeStore.update(
      target.organizationId,
      target.context.user_id,
      articleId.value,
      expectedVersion.value.version,
      {
        ...(title === undefined ? {} : { title }),
        ...(input.value.body === undefined ? {} : { body: input.value.body }),
      }
    ),
    200
  );
}

export async function deleteKnowledgeArticleRoute(
  request: Request,
  rawId: string
): Promise<NextResponse> {
  const target = await resolveKnowledgeMutationRoute(request);
  if (!target.available) return target.response;
  const manager = requireManager(target);
  if (manager) return manager;
  const articleId = parseArticleId(rawId);
  if (!articleId.ok) return articleId.response;
  const expectedVersion = expectedArticleVersion(request);
  if (!expectedVersion.ok) return expectedVersion.response;

  if (target.backend === 'go-api') {
    return forwardAuthenticatedGoApi(request, {
      method: 'DELETE',
      path: `knowledge-articles/${articleId.value}`,
      accessToken: target.accessToken,
      organizationId: String(target.organizationId),
      ifMatch: expectedVersion.value.etag,
    });
  }
  return mockDeleteResponse(
    mockKnowledgeStore.delete(target.organizationId, articleId.value, expectedVersion.value.version)
  );
}

/** Reads: feature availability, then authentication, then the organization context. */
async function resolveKnowledgeReadRoute(request: Request): Promise<KnowledgeTarget> {
  const resolution = resolveKnowledgeRoute();
  if (!resolution.available) return resolution;
  const authentication = await authenticateKnowledgeBackend(resolution.backend);
  if (!authentication.authenticated) return { available: false, response: authentication.response };
  return resolveOrganizationTarget(authentication, request);
}

/**
 * Writes: feature availability, then the same-origin check, then authentication, then the
 * organization context. Handlers read the body only after this resolves.
 */
async function resolveKnowledgeMutationRoute(request: Request): Promise<KnowledgeTarget> {
  const resolution = resolveKnowledgeRoute();
  if (!resolution.available) return resolution;
  const guard = guardSameOriginMutation(request);
  if (guard) return { available: false, response: privateKnowledgeResponse(guard) };
  const authentication = await authenticateKnowledgeBackend(resolution.backend);
  if (!authentication.authenticated) return { available: false, response: authentication.response };
  return resolveOrganizationTarget(authentication, request);
}

async function authenticateKnowledgeBackend(
  backend: KnowledgeBackend
): Promise<AuthenticatedKnowledgeBackend> {
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

function resolveOrganizationTarget(
  authentication: Authentication,
  request: Request
): KnowledgeTarget {
  const rawOrganizationId = request.headers.get('organization-id');
  if (rawOrganizationId === null) {
    return organizationFailure(
      ApiErrorCode.ORGANIZATION_CONTEXT_REQUIRED,
      'Organization context is required'
    );
  }
  const organizationId = parseOrganizationId(rawOrganizationId);
  if (!organizationId) {
    return organizationFailure(
      ApiErrorCode.ORGANIZATION_CONTEXT_INVALID,
      'Organization context is invalid'
    );
  }
  if (authentication.backend === 'go-api') {
    return {
      available: true,
      backend: 'go-api',
      organizationId,
      accessToken: authentication.accessToken,
    };
  }
  const context = mockOrganizationStore.resolveContext(authentication.user, organizationId);
  if (!context) {
    return {
      available: false,
      response: apiErrorResponse({
        status: 404,
        errorCode: ApiErrorCode.ORGANIZATION_NOT_FOUND,
        message: 'Organization not found',
        headers: privateKnowledgeHeaders(),
      }),
    };
  }
  return { available: true, backend: 'mock', organizationId, context };
}

/** The mock enforces the API's owner/admin write rule; the API enforces it for the adapter. */
function requireManager(target: Extract<KnowledgeTarget, { available: true }>) {
  if (target.backend !== 'mock') return null;
  const role = target.context.role;
  return role === 'owner' || role === 'admin'
    ? null
    : apiErrorResponse({
        status: 403,
        errorCode: ApiErrorCode.PERMISSION_DENIED,
        message: 'Operation forbidden',
        headers: privateKnowledgeHeaders(),
      });
}

async function articleBody(
  request: Request,
  mode: 'create' | 'update'
): Promise<Parsed<{ title?: string; body?: string }>> {
  const payload = await readJsonBody(request, maxArticleBodyBytes);
  if (!payload.ok) {
    return {
      ok: false,
      response: privateKnowledgeResponse(apiJsonBodyErrorResponse(payload.error)),
    };
  }
  const value = payload.data;
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return { ok: false, response: invalidInput('Malformed JSON body') };
  }
  const fields = value as Record<string, unknown>;
  const errors: Record<string, string[]> = {};
  for (const key of Object.keys(fields)) {
    if (key !== 'title' && key !== 'body') errors[key] = ['The field is not allowed'];
  }
  const { title, body } = fields;
  if (title === undefined) {
    if (mode === 'create') errors.title = ['The title field is required'];
  } else if (
    typeof title !== 'string' ||
    (mode === 'create' && title === '') ||
    characterCount(title) > KNOWLEDGE_TITLE_MAX_LENGTH
  ) {
    errors.title = ['The title field is invalid'];
  }
  if (
    body !== undefined &&
    (typeof body !== 'string' || characterCount(body) > KNOWLEDGE_BODY_MAX_LENGTH)
  ) {
    errors.body = ['The body field is invalid'];
  }
  if (Object.keys(errors).length > 0) return { ok: false, response: validationFailed(errors) };
  return {
    ok: true,
    value: {
      ...(typeof title === 'string' ? { title } : {}),
      ...(typeof body === 'string' ? { body } : {}),
    },
  };
}

function expectedArticleVersion(request: Request): Parsed<{ version: number; etag: string }> {
  const etag = request.headers.get('if-match');
  if (etag === null) {
    return {
      ok: false,
      response: apiErrorResponse({
        status: 428,
        errorCode: ApiErrorCode.KNOWLEDGE_PRECONDITION_REQUIRED,
        message: 'Knowledge article version required',
        headers: privateKnowledgeHeaders(),
      }),
    };
  }
  const match = articleVersionPattern.exec(etag);
  const version = match ? Number(match[1]) : Number.NaN;
  if (!Number.isSafeInteger(version)) {
    return { ok: false, response: invalidInput('Invalid If-Match header') };
  }
  return { ok: true, value: { version, etag } };
}

function parseArticleId(value: string): Parsed<number> {
  const parsed = knowledgeRouteIdSchema.safeParse(value);
  const id = parsed.success ? Number(parsed.data) : Number.NaN;
  return Number.isSafeInteger(id)
    ? { ok: true, value: id }
    : { ok: false, response: invalidInput('Invalid knowledge article ID') };
}

/** Mirrors the API: a missing or malformed size uses the default, a large one is capped. */
function parsePerPage(value: string | null): number {
  if (value === null || !/^\d{1,6}$/u.test(value)) return defaultPerPage;
  const parsed = Number(value);
  if (parsed < 1) return defaultPerPage;
  return Math.min(parsed, KNOWLEDGE_PAGE_MAX_SIZE);
}

function mockWriteResponse(result: MockKnowledgeWriteResult, status: 200 | 201): NextResponse {
  if (result.ok) return articleResponse(result.article, status);
  if (result.error === 'capacity') return unavailableResponse('Mock knowledge store is full');
  return result.error === 'not_found' ? articleNotFound() : versionConflict();
}

function mockDeleteResponse(result: MockKnowledgeDeleteResult): NextResponse {
  if (result.ok) return apiNoContentResponse(privateKnowledgeHeaders());
  return result.error === 'not_found' ? articleNotFound() : versionConflict();
}

function articleResponse(article: KnowledgeArticle, status = 200): NextResponse {
  const headers = privateKnowledgeHeaders();
  headers.set('etag', `"knowledge-article-v${article.version}"`);
  return apiSuccessResponse(article, { status, headers });
}

/**
 * The shared helpers have no cursor-page envelope, so the mock writes the contract's
 * `KnowledgeArticlePageResponse` shape here: the success envelope plus `meta`.
 */
function knowledgePageResponse(page: KnowledgeArticlePage): NextResponse {
  return NextResponse.json(
    { code: 0, message: 'success', data: page.items, meta: page.meta },
    { headers: privateKnowledgeHeaders() }
  );
}

function currentEnvironment(): KnowledgeRouteEnvironment {
  return {
    adapterEnabled: serverEnv.API_ADAPTER_ENABLED,
    featureEnabled: isWebFeatureEnabled('knowledge'),
    mockBffEnabled: serverEnv.MOCK_BFF_ENABLED,
    nodeEnv: env.NODE_ENV,
  };
}

function privateKnowledgeHeaders(): Headers {
  return privateNoStoreHeaders(undefined, ['Cookie', 'Organization-Id']);
}

function organizationFailure(
  errorCode:
    | typeof ApiErrorCode.ORGANIZATION_CONTEXT_REQUIRED
    | typeof ApiErrorCode.ORGANIZATION_CONTEXT_INVALID,
  message: string
): KnowledgeTarget {
  return {
    available: false,
    response: apiErrorResponse({
      status: 400,
      errorCode,
      message,
      headers: privateKnowledgeHeaders(),
    }),
  };
}

function invalidInput(message: string): NextResponse {
  return apiErrorResponse({
    status: 400,
    errorCode: ApiErrorCode.COMMON_INVALID_INPUT,
    message,
    headers: privateKnowledgeHeaders(),
  });
}

function validationFailed(errors: Record<string, string[]>): NextResponse {
  return apiErrorResponse({
    status: 422,
    errorCode: ApiErrorCode.COMMON_VALIDATION_FAILED,
    message: 'Validation failed',
    errors,
    headers: privateKnowledgeHeaders(),
  });
}

function invalidArticle(): NextResponse {
  return apiErrorResponse({
    status: 422,
    errorCode: ApiErrorCode.KNOWLEDGE_INVALID_ARTICLE,
    message: 'Invalid knowledge article',
    headers: privateKnowledgeHeaders(),
  });
}

function articleNotFound(): NextResponse {
  return apiErrorResponse({
    status: 404,
    errorCode: ApiErrorCode.KNOWLEDGE_ARTICLE_NOT_FOUND,
    message: 'Knowledge article not found',
    headers: privateKnowledgeHeaders(),
  });
}

function versionConflict(): NextResponse {
  return apiErrorResponse({
    status: 412,
    errorCode: ApiErrorCode.KNOWLEDGE_VERSION_CONFLICT,
    message: 'Knowledge article version conflict',
    headers: privateKnowledgeHeaders(),
  });
}

function unauthenticated(): NextResponse {
  return apiErrorResponse({
    status: 401,
    errorCode: ApiErrorCode.AUTH_UNAUTHORIZED,
    message: 'Authentication required',
    headers: privateKnowledgeHeaders(),
  });
}

function unavailableResponse(message: string): NextResponse {
  return apiErrorResponse({
    status: 503,
    errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
    message,
    headers: privateKnowledgeHeaders(),
  });
}

function unavailable(message: string): KnowledgeRouteResolution {
  return {
    available: false,
    response: apiErrorResponse({
      status: 503,
      errorCode: ApiErrorCode.COMMON_SERVICE_UNAVAILABLE,
      message,
      headers: new Headers({ 'cache-control': 'no-store', pragma: 'no-cache' }),
    }),
  };
}
