import { readdirSync } from 'node:fs';
import { join, relative, sep } from 'node:path';
import { AxiosError, type AxiosAdapter, type InternalAxiosRequestConfig } from 'axios';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

/**
 * Parity between the mock BFF and the browser services.
 *
 * Route tests prove mock behavior and the per-feature contract tests prove the parsers accept
 * sample envelopes, but neither proves the mock emits what the services accept. Here every
 * feature service runs unchanged against the real `src/app/api` route handlers, so a mock
 * response that drifts from the service's Zod schema fails as an invalid response.
 */

const cookieStore = vi.hoisted(() => ({ get: vi.fn(), set: vi.fn(), delete: vi.fn() }));
const getSessionUser = vi.hoisted(() => vi.fn());

vi.mock('next/headers', () => ({
  cookies: async () => cookieStore,
  headers: async () => new Headers(),
}));
vi.mock('@/features/auth/server/session', async importOriginal => {
  const original = await importOriginal<typeof import('@/features/auth/server/session')>();
  return { ...original, getSessionUser };
});
vi.mock('@/http/request', async importOriginal => {
  const original = await importOriginal<typeof import('@/http/request')>();
  const client = original.createRequest({ baseURL: '/api', adapter: mockBffAdapter });
  return { ...original, request: client, default: client };
});

const APP_ORIGIN = 'https://app.example.com';
const ORGANIZATION_ID = 1;

type RouteHandler = (request: Request, context: { params: Promise<Record<string, string>> }) => Promise<Response>;
type RouteModule = Partial<Record<string, RouteHandler>>;

const apiDirectory = join(process.cwd(), 'src', 'app', 'api');
const routes = readdirSync(apiDirectory, { recursive: true, encoding: 'utf8' })
  .filter(file => file === 'route.ts' || file.endsWith(`${sep}route.ts`))
  .map(file => ({
    segments: relative(apiDirectory, join(apiDirectory, file, '..')).split(sep).filter(Boolean),
    load: () => import(/* @vite-ignore */ join(apiDirectory, file)) as Promise<RouteModule>,
  }));

function matchRoute(pathname: string) {
  const parts = pathname.replace(/^\/api\/?/, '').split('/').filter(Boolean);
  // Static segments win over dynamic ones, as in the Next.js router.
  const ranked = [...routes].sort(
    (left, right) =>
      left.segments.filter(segment => segment.startsWith('[')).length -
      right.segments.filter(segment => segment.startsWith('[')).length
  );
  for (const route of ranked) {
    if (route.segments.length !== parts.length) continue;
    const params: Record<string, string> = {};
    const matched = route.segments.every((segment, index) => {
      if (segment.startsWith('[') && segment.endsWith(']')) {
        params[segment.slice(1, -1)] = decodeURIComponent(parts[index]);
        return true;
      }
      return segment === parts[index];
    });
    if (matched) return { route, params };
  }
  return null;
}

function requestURL(config: InternalAxiosRequestConfig): URL {
  const url = new URL(`${APP_ORIGIN}${config.baseURL ?? ''}${config.url ?? ''}`);
  for (const [key, value] of Object.entries((config.params ?? {}) as Record<string, unknown>)) {
    if (value !== undefined && value !== null) url.searchParams.set(key, String(value));
  }
  return url;
}

async function mockBffAdapter(config: InternalAxiosRequestConfig): ReturnType<AxiosAdapter> {
  const url = requestURL(config);
  const method = (config.method ?? 'get').toUpperCase();
  const match = matchRoute(url.pathname);
  const handler = match ? (await match.route.load())[method] : undefined;
  if (!match || !handler) {
    throw new Error(`mock BFF has no ${method} handler for ${url.pathname}`);
  }

  const headers = new Headers();
  for (const [key, value] of Object.entries(config.headers.toJSON())) {
    if (typeof value === 'string' || typeof value === 'number') headers.set(key, String(value));
  }
  if (method !== 'GET') {
    headers.set('origin', APP_ORIGIN);
    headers.set('sec-fetch-site', 'same-origin');
  }

  const response = await handler(
    new Request(url, { method, headers, body: method === 'GET' ? undefined : config.data }),
    { params: Promise.resolve(match.params) }
  );
  const axiosResponse = {
    config,
    data: await response.text(),
    headers: Object.fromEntries(response.headers.entries()),
    status: response.status,
    statusText: response.statusText,
  };
  if (response.status >= 400) {
    throw new AxiosError(
      `mock BFF returned ${response.status}`,
      AxiosError.ERR_BAD_RESPONSE,
      config,
      null,
      axiosResponse
    );
  }
  return axiosResponse;
}

const originalEnv = { ...process.env };
const managedKeys = [
  'API_ADAPTER_ENABLED',
  'MOCK_BFF_ENABLED',
  'NEXT_PUBLIC_API_URL',
  'NEXT_PUBLIC_APP_URL',
  'NEXT_PUBLIC_OPTIONAL_FEATURES',
] as const;

describe('mock BFF and browser service parity', () => {
  beforeEach(() => {
    vi.resetModules();
    cookieStore.get.mockReset();
    cookieStore.set.mockReset();
    getSessionUser.mockReset();
    getSessionUser.mockResolvedValue({
      id: 'demo-admin',
      email: 'admin@example.com',
      name: 'Admin User',
    });
    delete process.env.API_ADAPTER_ENABLED;
    process.env.MOCK_BFF_ENABLED = 'true';
    process.env.NEXT_PUBLIC_API_URL = '/api';
    process.env.NEXT_PUBLIC_APP_URL = APP_ORIGIN;
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES =
      'organization,permission,setting,usage,webhook,notification,asset';
  });

  afterEach(() => {
    vi.resetModules();
    for (const key of managedKeys) {
      delete process.env[key];
      if (originalEnv[key] !== undefined) process.env[key] = originalEnv[key];
    }
  });

  it('api keys', async () => {
    const { apiKeyService } = await import('@/features/api-key/services/api-key-service');

    const created = await apiKeyService.create({ name: 'parity', scopes: ['luas:read'] });
    const page = await apiKeyService.list();
    expect(page.items.map(item => item.id)).toContain(created.api_key.id);
    await apiKeyService.revoke(created.api_key.id);
  });

  it('organizations', async () => {
    const { organizationService } = await import(
      '@/features/organization/services/organization-service'
    );

    const page = await organizationService.list();
    expect(page.items.length).toBeGreaterThan(0);
    await organizationService.get(ORGANIZATION_ID);
    await organizationService.resolveContext(ORGANIZATION_ID);
    const members = await organizationService.listMembers(ORGANIZATION_ID);
    expect(members.items.length).toBeGreaterThan(0);
    await organizationService.invite(ORGANIZATION_ID, {
      email: 'parity@example.com',
      role: 'member',
    });
    const invitations = await organizationService.listInvitations(ORGANIZATION_ID);
    expect(invitations.items.length).toBeGreaterThan(0);
  });

  it('permissions', async () => {
    const [{ permissionService }, { organizationService }] = await Promise.all([
      import('@/features/permission/services/permission-service'),
      import('@/features/organization/services/organization-service'),
    ]);

    await permissionService.effective(ORGANIZATION_ID);
    await permissionService.catalog(ORGANIZATION_ID);
    await permissionService.listRoles(ORGANIZATION_ID);
    const members = await organizationService.listMembers(ORGANIZATION_ID);
    await permissionService.memberRoles(ORGANIZATION_ID, members.items[0].id);
  });

  it('settings', async () => {
    const { settingService } = await import('@/features/setting/services/setting-service');

    await settingService.publicApp();
    await settingService.user();
    await settingService.organization(ORGANIZATION_ID);
  });

  it('usage', async () => {
    const { usageService } = await import('@/features/usage/services/usage-service');

    await usageService.user();
    await usageService.organization(ORGANIZATION_ID);
  });

  it('notifications', async () => {
    const { notificationService } = await import(
      '@/features/notification/services/notification-service'
    );

    await notificationService.list();
    await notificationService.status();
    const preference = await notificationService.preference();
    await notificationService.replacePreference(preference);
  });

  it('assets', async () => {
    const { assetService } = await import('@/features/asset/services/asset-service');

    await assetService.list();
  });

  it('webhooks', async () => {
    const { webhookService } = await import('@/features/webhook/services/webhook-service');

    await webhookService.eventTypes(ORGANIZATION_ID);
    const created = await webhookService.create(ORGANIZATION_ID, {
      name: 'parity',
      url: 'https://hooks.example.com/luas',
      event_types: ['webhook.test'],
    });
    const page = await webhookService.endpoints(ORGANIZATION_ID);
    expect(page.items.map(item => item.id)).toContain(created.endpoint.id);
    await webhookService.deliveries(ORGANIZATION_ID);
  });
});
