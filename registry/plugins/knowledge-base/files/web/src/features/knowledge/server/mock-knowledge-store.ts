import 'server-only';

import {
  KNOWLEDGE_EXCERPT_MAX_LENGTH,
  KNOWLEDGE_SEARCH_MAX_RESULTS,
} from '@/features/knowledge/schemas';
import type {
  KnowledgeArticle,
  KnowledgeArticlePage,
  KnowledgeSearchHit,
} from '@/features/knowledge/types';

/**
 * Development-only knowledge articles, isolated per organization. It mirrors the browser-facing
 * behavior of the Go knowledge starter: keyset pages newest first, monotonic versions,
 * version-checked writes, and substring search over title and body. It is not a search engine.
 */
const maxMockOrganizations = 1_000;
const maxMockArticlesPerOrganization = 1_000;

interface OrganizationState {
  nextId: number;
  articles: Map<number, KnowledgeArticle>;
}

export type MockKnowledgeWriteResult =
  | { ok: true; article: KnowledgeArticle }
  | { ok: false; error: 'not_found' | 'version_conflict' | 'capacity' };

export type MockKnowledgeDeleteResult =
  { ok: true } | { ok: false; error: 'not_found' | 'version_conflict' };

const organizations = new Map<number, OrganizationState>();

export const mockKnowledgeStore = {
  list(organizationId: number, afterId: number | null, perPage: number): KnowledgeArticlePage {
    const ordered = [...stateFor(organizationId).articles.values()]
      .filter(article => afterId === null || article.id < afterId)
      .sort((left, right) => right.id - left.id);
    const page = ordered.slice(0, perPage);
    const hasMore = ordered.length > perPage;
    const last = page.at(-1);
    return {
      items: page.map(({ id, title, version, updated_at }) => ({ id, title, version, updated_at })),
      meta: {
        per_page: perPage,
        has_more: hasMore,
        next_cursor: hasMore && last ? encodeKnowledgeCursor(last.id) : null,
      },
    };
  },

  get(organizationId: number, articleId: number): KnowledgeArticle | null {
    return stateFor(organizationId).articles.get(articleId) ?? null;
  },

  create(
    organizationId: number,
    actorId: number,
    input: { title: string; body: string }
  ): MockKnowledgeWriteResult {
    const state = stateFor(organizationId);
    if (state.articles.size >= maxMockArticlesPerOrganization) {
      return { ok: false, error: 'capacity' };
    }
    const now = new Date().toISOString();
    const article: KnowledgeArticle = {
      id: state.nextId,
      title: input.title,
      body: input.body,
      version: 1,
      created_by: actorId,
      updated_by: actorId,
      created_at: now,
      updated_at: now,
    };
    state.nextId += 1;
    state.articles.set(article.id, article);
    return { ok: true, article };
  },

  update(
    organizationId: number,
    actorId: number,
    articleId: number,
    expectedVersion: number,
    changes: { title?: string; body?: string }
  ): MockKnowledgeWriteResult {
    const state = stateFor(organizationId);
    const current = state.articles.get(articleId);
    if (!current) return { ok: false, error: 'not_found' };
    if (current.version !== expectedVersion) return { ok: false, error: 'version_conflict' };
    const article: KnowledgeArticle = {
      ...current,
      ...(changes.title === undefined ? {} : { title: changes.title }),
      ...(changes.body === undefined ? {} : { body: changes.body }),
      version: current.version + 1,
      updated_by: actorId,
      updated_at: new Date().toISOString(),
    };
    state.articles.set(articleId, article);
    return { ok: true, article };
  },

  delete(
    organizationId: number,
    articleId: number,
    expectedVersion: number
  ): MockKnowledgeDeleteResult {
    const state = stateFor(organizationId);
    const current = state.articles.get(articleId);
    if (!current) return { ok: false, error: 'not_found' };
    if (current.version !== expectedVersion) return { ok: false, error: 'version_conflict' };
    state.articles.delete(articleId);
    return { ok: true };
  },

  /** Case-insensitive substring search; title matches rank above body matches, then newest. */
  search(organizationId: number, query: string): KnowledgeSearchHit[] {
    const needle = query.trim().toLocaleLowerCase();
    return [...stateFor(organizationId).articles.values()]
      .map(article => ({
        article,
        rank: article.title.toLocaleLowerCase().includes(needle)
          ? 2
          : article.body.toLocaleLowerCase().includes(needle)
            ? 1
            : 0,
      }))
      .filter(({ rank }) => rank > 0)
      .sort((left, right) => right.rank - left.rank || right.article.id - left.article.id)
      .slice(0, KNOWLEDGE_SEARCH_MAX_RESULTS)
      .map(({ article }) => ({
        id: article.id,
        title: article.title,
        excerpt: Array.from(article.body).slice(0, KNOWLEDGE_EXCERPT_MAX_LENGTH).join(''),
        version: article.version,
        updated_at: article.updated_at,
      }));
  },

  /** Test seam: forget every organization's articles. */
  reset(): void {
    organizations.clear();
  },
};

/** Cursors are opaque to clients; like the API, the mock encodes the last article ID. */
export function encodeKnowledgeCursor(articleId: number): string {
  return Buffer.from(String(articleId), 'utf8').toString('base64url');
}

export function decodeKnowledgeCursor(cursor: string): number | null {
  if (cursor.length < 1 || cursor.length > 32 || !/^[A-Za-z0-9_-]+$/u.test(cursor)) return null;
  const raw = Buffer.from(cursor, 'base64url').toString('utf8');
  if (!/^[1-9]\d{0,15}$/u.test(raw)) return null;
  const id = Number(raw);
  return Number.isSafeInteger(id) ? id : null;
}

function stateFor(organizationId: number): OrganizationState {
  const existing = organizations.get(organizationId);
  if (existing) return existing;
  if (organizations.size >= maxMockOrganizations) {
    const oldest = organizations.keys().next().value;
    if (oldest !== undefined) organizations.delete(oldest);
  }
  const created: OrganizationState = { nextId: 1, articles: new Map() };
  organizations.set(organizationId, created);
  return created;
}
