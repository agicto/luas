import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { parseOptionalWebFeatures } from '@/config/optional-features';

const cookieStore = vi.hoisted(() => ({ get: vi.fn(), set: vi.fn() }));
const getSessionUser = vi.hoisted(() => vi.fn());

vi.mock('next/headers', () => ({
  cookies: async () => cookieStore,
  headers: async () => new Headers(),
}));
vi.mock('@/features/auth/server/session', async importOriginal => {
  const original = await importOriginal<typeof import('@/features/auth/server/session')>();
  return { ...original, getSessionUser };
});

const originalEnv = { ...process.env };
const managedKeys = [
  'API_ADAPTER_ENABLED',
  'API_UPSTREAM_MAX_RESPONSE_BYTES',
  'API_UPSTREAM_TIMEOUT_MS',
  'API_UPSTREAM_URL',
  'MOCK_BFF_ENABLED',
  'NEXT_PUBLIC_API_URL',
  'NEXT_PUBLIC_APP_URL',
  'NEXT_PUBLIC_OPTIONAL_FEATURES',
] as const;

type KnowledgeRoute = typeof import('@/features/knowledge/server/knowledge-route');

describe('knowledge browser route boundary', () => {
  let fetchMock: ReturnType<typeof vi.fn<typeof fetch>>;

  beforeEach(() => {
    vi.resetModules();
    fetchMock = vi.fn<typeof fetch>();
    vi.stubGlobal('fetch', fetchMock);
    cookieStore.get.mockReset();
    cookieStore.set.mockReset();
    getSessionUser.mockReset();
    process.env.NEXT_PUBLIC_API_URL = '/api';
    process.env.NEXT_PUBLIC_APP_URL = 'https://app.example.com';
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'organization,knowledge';
    process.env.API_UPSTREAM_TIMEOUT_MS = '5000';
    process.env.API_UPSTREAM_MAX_RESPONSE_BYTES = '1048576';
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.resetModules();
    for (const key of managedKeys) {
      delete process.env[key];
      if (originalEnv[key] !== undefined) process.env[key] = originalEnv[key];
    }
  });

  it('selects knowledge only with organization ownership', () => {
    expect(parseOptionalWebFeatures('organization,knowledge')).toEqual([
      'organization',
      'knowledge',
    ]);
    expect(() => parseOptionalWebFeatures('knowledge')).toThrow('requires "organization"');
  });

  it('stays unavailable when the feature is not selected', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    process.env.NEXT_PUBLIC_OPTIONAL_FEATURES = 'organization';
    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const route = await loadRoute();

    const response = await route.listKnowledgeArticlesRoute(
      request('/api/knowledge-articles', { headers: organization() })
    );

    expect(response.status).toBe(503);
    await expect(response.json()).resolves.toMatchObject({
      error_code: 'COMMON.SERVICE_UNAVAILABLE',
    });
    expect(getSessionUser).not.toHaveBeenCalled();
  });

  it('creates, pages newest first, reads with an ETag, and searches substrings', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const route = await loadRoute();

    const first = await create(route, {
      title: '  Refund policy  ',
      body: '退款需要在七天内申请。',
    });
    expect(first.status).toBe(201);
    expect(first.headers.get('etag')).toBe('"knowledge-article-v1"');
    expect(first.headers.get('cache-control')).toBe('private, no-store');
    expect(first.headers.get('vary')).toContain('Organization-Id');
    const created = (await first.json()).data;
    expect(created).toMatchObject({
      title: 'Refund policy',
      version: 1,
      body: '退款需要在七天内申请。',
    });
    const second = (await (await create(route, { title: 'Onboarding' })).json()).data;
    expect(second.body).toBe('');

    const page = await route.listKnowledgeArticlesRoute(
      request('/api/knowledge-articles?per_page=1', { headers: organization() })
    );
    const pageBody = await page.json();
    expect(pageBody).toMatchObject({
      code: 0,
      data: [{ id: second.id, title: 'Onboarding' }],
      meta: { per_page: 1, has_more: true },
    });
    expect(pageBody.data[0]).not.toHaveProperty('body');
    const next = await route.listKnowledgeArticlesRoute(
      request(`/api/knowledge-articles?per_page=1&cursor=${pageBody.meta.next_cursor}`, {
        headers: organization(),
      })
    );
    await expect(next.json()).resolves.toMatchObject({
      data: [{ id: created.id }],
      meta: { has_more: false, next_cursor: null },
    });

    const read = await route.getKnowledgeArticleRoute(
      request(`/api/knowledge-articles/${created.id}`, { headers: organization() }),
      String(created.id)
    );
    expect(read.status).toBe(200);
    expect(read.headers.get('etag')).toBe('"knowledge-article-v1"');

    const search = await route.searchKnowledgeArticlesRoute(
      request(`/api/knowledge-search?q=${encodeURIComponent('七天')}`, { headers: organization() })
    );
    await expect(search.json()).resolves.toMatchObject({
      data: [{ id: created.id, excerpt: '退款需要在七天内申请。' }],
    });
  });

  it('isolates organizations and lets members read but only owners and admins write', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const route = await loadRoute();
    const created = (await (await create(route, { title: 'Private to one' })).json()).data;

    getSessionUser.mockResolvedValue(mockUser('demo-member'));
    const read = await route.getKnowledgeArticleRoute(
      request(`/api/knowledge-articles/${created.id}`, { headers: organization() }),
      String(created.id)
    );
    expect(read.status).toBe(200);
    const forbidden = await create(route, { title: 'Member write' });
    expect(forbidden.status).toBe(403);
    await expect(forbidden.json()).resolves.toMatchObject({ error_code: 'PERMISSION.DENIED' });

    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const otherOrganization = await route.getKnowledgeArticleRoute(
      request(`/api/knowledge-articles/${created.id}`, { headers: organization('999') }),
      String(created.id)
    );
    expect(otherOrganization.status).toBe(404);
    await expect(otherOrganization.json()).resolves.toMatchObject({
      error_code: 'ORGANIZATION.NOT_FOUND',
    });

    const missingContext = await route.listKnowledgeArticlesRoute(
      request('/api/knowledge-articles')
    );
    expect(missingContext.status).toBe(400);
    await expect(missingContext.json()).resolves.toMatchObject({
      error_code: 'ORGANIZATION.CONTEXT_REQUIRED',
    });
  });

  it('requires If-Match, rejects stale versions, and deletes with the current version', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const route = await loadRoute();
    const created = (await (await create(route, { title: 'Versioned' })).json()).data;
    const path = `/api/knowledge-articles/${created.id}`;

    const missing = await route.updateKnowledgeArticleRoute(
      request(path, {
        method: 'PATCH',
        body: JSON.stringify({ title: 'Changed' }),
        headers: organization(),
      }),
      String(created.id)
    );
    expect(missing.status).toBe(428);
    await expect(missing.json()).resolves.toMatchObject({
      error_code: 'KNOWLEDGE.PRECONDITION_REQUIRED',
    });

    const malformed = await route.updateKnowledgeArticleRoute(
      request(path, {
        method: 'PATCH',
        body: JSON.stringify({ title: 'Changed' }),
        headers: { ...organization(), 'If-Match': 'W/"knowledge-article-v1"' },
      }),
      String(created.id)
    );
    expect(malformed.status).toBe(400);

    const updated = await route.updateKnowledgeArticleRoute(
      request(path, {
        method: 'PATCH',
        body: JSON.stringify({ title: 'Changed', body: 'New body' }),
        headers: { ...organization(), 'If-Match': '"knowledge-article-v1"' },
      }),
      String(created.id)
    );
    expect(updated.status).toBe(200);
    expect(updated.headers.get('etag')).toBe('"knowledge-article-v2"');

    const stale = await route.deleteKnowledgeArticleRoute(
      request(path, {
        method: 'DELETE',
        headers: { ...organization(), 'If-Match': '"knowledge-article-v1"' },
      }),
      String(created.id)
    );
    expect(stale.status).toBe(412);
    await expect(stale.json()).resolves.toMatchObject({
      error_code: 'KNOWLEDGE.VERSION_CONFLICT',
    });

    const removed = await route.deleteKnowledgeArticleRoute(
      request(path, {
        method: 'DELETE',
        headers: { ...organization(), 'If-Match': '"knowledge-article-v2"' },
      }),
      String(created.id)
    );
    expect(removed.status).toBe(204);
    const gone = await route.getKnowledgeArticleRoute(
      request(path, { headers: organization() }),
      String(created.id)
    );
    expect(gone.status).toBe(404);
    await expect(gone.json()).resolves.toMatchObject({
      error_code: 'KNOWLEDGE.ARTICLE_NOT_FOUND',
    });
  });

  it('validates input like the API', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    getSessionUser.mockResolvedValue(mockUser('demo-admin'));
    const route = await loadRoute();

    const blank = await create(route, { title: '   ' });
    expect(blank.status).toBe(422);
    await expect(blank.json()).resolves.toMatchObject({
      error_code: 'KNOWLEDGE.INVALID_ARTICLE',
    });
    const missing = await create(route, { body: 'no title' });
    expect(missing.status).toBe(422);
    await expect(missing.json()).resolves.toMatchObject({
      error_code: 'COMMON.VALIDATION_FAILED',
      errors: { title: expect.any(Array) },
    });
    const tooLong = await create(route, { title: '题'.repeat(201) });
    expect(tooLong.status).toBe(422);
    const unknownField = await create(route, { title: 'ok', pinned: true });
    expect(unknownField.status).toBe(422);

    const created = (await (await create(route, { title: 'Target' })).json()).data;
    const empty = await route.updateKnowledgeArticleRoute(
      request(`/api/knowledge-articles/${created.id}`, {
        method: 'PATCH',
        body: '{}',
        headers: { ...organization(), 'If-Match': '"knowledge-article-v1"' },
      }),
      String(created.id)
    );
    expect(empty.status).toBe(422);
    await expect(empty.json()).resolves.toMatchObject({ error_code: 'KNOWLEDGE.INVALID_ARTICLE' });

    const badCursor = await route.listKnowledgeArticlesRoute(
      request('/api/knowledge-articles?cursor=%2F%2F', { headers: organization() })
    );
    expect(badCursor.status).toBe(400);
    const badId = await route.getKnowledgeArticleRoute(
      request('/api/knowledge-articles/abc', { headers: organization() }),
      'abc'
    );
    expect(badId.status).toBe(400);
    const blankQuery = await route.searchKnowledgeArticlesRoute(
      request('/api/knowledge-search?q=%20%20', { headers: organization() })
    );
    expect(blankQuery.status).toBe(400);
    const missingQuery = await route.searchKnowledgeArticlesRoute(
      request('/api/knowledge-search', { headers: organization() })
    );
    expect(missingQuery.status).toBe(422);
  });

  it('rejects cross-origin writes before authentication or body parsing', async () => {
    process.env.MOCK_BFF_ENABLED = 'true';
    const route = await loadRoute();

    const response = await route.createKnowledgeArticleRoute(
      request('/api/knowledge-articles', {
        method: 'POST',
        body: '{',
        origin: 'https://evil.example',
        headers: organization(),
      })
    );

    expect(response.status).toBe(403);
    expect(response.headers.get('cache-control')).toBe('private, no-store');
    expect(getSessionUser).not.toHaveBeenCalled();
  });

  it('forwards only fixed production paths, bounded queries, and the If-Match validator', async () => {
    process.env.API_ADAPTER_ENABLED = 'true';
    process.env.API_UPSTREAM_URL = 'https://api.example.com/v1';
    cookieStore.get.mockReturnValue({ value: 'A'.repeat(43) });
    fetchMock
      .mockResolvedValueOnce(
        Response.json({
          code: 0,
          message: 'success',
          data: [],
          meta: { per_page: 100, has_more: false, next_cursor: null },
        })
      )
      .mockResolvedValueOnce(Response.json({ code: 0, message: 'success', data: [] }))
      .mockResolvedValueOnce(
        Response.json(
          {
            code: 412,
            error_code: 'KNOWLEDGE.VERSION_CONFLICT',
            message: 'Knowledge article version conflict',
          },
          { status: 412 }
        )
      );
    const route = await loadRoute();

    await route.listKnowledgeArticlesRoute(
      request('/api/knowledge-articles?per_page=500&cursor=MTI&extra=1', {
        headers: organization(),
      })
    );
    await route.searchKnowledgeArticlesRoute(
      request(`/api/knowledge-search?q=${encodeURIComponent('退款 policy')}`, {
        headers: organization(),
      })
    );
    const conflict = await route.updateKnowledgeArticleRoute(
      request('/api/knowledge-articles/12', {
        method: 'PATCH',
        body: JSON.stringify({ title: 'Changed' }),
        headers: { ...organization(), 'If-Match': '"knowledge-article-v3"' },
      }),
      '12'
    );

    const listURL = new URL(String(fetchMock.mock.calls[0][0]));
    expect(`${listURL.origin}${listURL.pathname}`).toBe(
      'https://api.example.com/v1/knowledge-articles'
    );
    expect(Object.fromEntries(listURL.searchParams)).toEqual({ per_page: '100', cursor: 'MTI' });
    expect(new Headers(fetchMock.mock.calls[0][1]?.headers).get('organization-id')).toBe('1');
    const searchURL = new URL(String(fetchMock.mock.calls[1][0]));
    expect(searchURL.pathname).toBe('/v1/knowledge-search');
    expect(searchURL.searchParams.get('q')).toBe('退款 policy');
    expect(String(fetchMock.mock.calls[2][0])).toBe(
      'https://api.example.com/v1/knowledge-articles/12'
    );
    expect(new Headers(fetchMock.mock.calls[2][1]?.headers).get('if-match')).toBe(
      '"knowledge-article-v3"'
    );
    // The adapter keeps the plugin's error code because the plugin registers it in ApiErrorCode.
    expect(conflict.status).toBe(412);
    await expect(conflict.json()).resolves.toMatchObject({
      error_code: 'KNOWLEDGE.VERSION_CONFLICT',
    });
  });
});

async function loadRoute(): Promise<KnowledgeRoute> {
  const route = await import('@/features/knowledge/server/knowledge-route');
  const store = await import('@/features/knowledge/server/mock-knowledge-store');
  store.mockKnowledgeStore.reset();
  return route;
}

function create(route: KnowledgeRoute, body: Record<string, unknown>) {
  return route.createKnowledgeArticleRoute(
    request('/api/knowledge-articles', {
      method: 'POST',
      body: JSON.stringify(body),
      headers: organization(),
    })
  );
}

function organization(id = '1'): Record<string, string> {
  return { 'Organization-Id': id };
}

function request(
  path: string,
  options: {
    method?: string;
    body?: string;
    origin?: string;
    headers?: Record<string, string>;
  } = {}
): Request {
  const result = new Request(`https://app.example.com${path}`, {
    method: options.method ?? 'GET',
    body: options.body,
    headers: options.headers,
  });
  if (options.body) result.headers.set('content-type', 'application/json');
  if (options.method && options.method !== 'GET') {
    result.headers.set('origin', options.origin ?? 'https://app.example.com');
    result.headers.set('sec-fetch-site', options.origin ? 'cross-site' : 'same-origin');
  }
  return result;
}

function mockUser(id: string) {
  return { id, email: `${id}@example.com`, name: id };
}
