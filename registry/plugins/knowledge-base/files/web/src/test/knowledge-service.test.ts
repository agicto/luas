import { describe, expect, it } from 'vitest';

import {
  characterCount,
  createKnowledgeArticleSchema,
  knowledgeArticlePageEnvelopeSchema,
  knowledgeSearchQuerySchema,
  knowledgeSearchResultSchema,
  updateKnowledgeArticleSchema,
} from '@/features/knowledge/schemas';
import {
  knowledgeArticleETag,
  parseArticle,
} from '@/features/knowledge/services/knowledge-service';
import { resolveKnowledgeErrorKey } from '@/features/knowledge/utils/knowledge-error';
import { ApiErrorCode, ClientErrorCode } from '@/http/codes';
import { ApiError } from '@/http/request';

describe('knowledge service boundary', () => {
  it('accepts contract-shaped articles and rejects drift', () => {
    expect(parseArticle(article(), 7)).toMatchObject({ id: 7, created_by: null });
    expect(() => parseArticle(article(), 8)).toThrow(ApiError);
    expect(() => parseArticle({ ...article(), extra: true })).toThrow(ApiError);
    expect(() => parseArticle({ ...article(), version: 0 })).toThrow(ApiError);
    expect(() => parseArticle({ ...article(), title: '' })).toThrow(ApiError);
  });

  it('accepts cursor pages and bounded search results', () => {
    expect(
      knowledgeArticlePageEnvelopeSchema.safeParse({
        code: 0,
        message: 'success',
        data: [{ id: 1, title: 'A', version: 1, updated_at: '2026-10-11T08:00:00Z' }],
        meta: { per_page: 15, has_more: true, next_cursor: 'MQ' },
      }).success
    ).toBe(true);
    expect(
      knowledgeArticlePageEnvelopeSchema.safeParse({
        code: 0,
        message: 'success',
        data: [{ id: 1, title: 'A', body: 'leak', version: 1, updated_at: '2026-10-11T08:00:00Z' }],
        meta: { per_page: 15, has_more: false, next_cursor: null },
      }).success
    ).toBe(false);
    const hit = {
      id: 1,
      title: 'A',
      excerpt: 'x',
      version: 1,
      updated_at: '2026-10-11T08:00:00Z',
    };
    expect(
      knowledgeSearchResultSchema.safeParse(Array.from({ length: 20 }, () => hit)).success
    ).toBe(true);
    expect(
      knowledgeSearchResultSchema.safeParse(Array.from({ length: 21 }, () => hit)).success
    ).toBe(false);
  });

  it('validates writes and queries by Unicode characters', () => {
    expect(characterCount('知识😀')).toBe(3);
    expect(createKnowledgeArticleSchema.safeParse({ title: '😀'.repeat(200) }).success).toBe(true);
    expect(createKnowledgeArticleSchema.safeParse({ title: '😀'.repeat(201) }).success).toBe(false);
    expect(createKnowledgeArticleSchema.safeParse({ title: '   ' }).success).toBe(false);
    expect(updateKnowledgeArticleSchema.safeParse({}).success).toBe(false);
    expect(updateKnowledgeArticleSchema.safeParse({ body: '' }).success).toBe(true);
    expect(knowledgeSearchQuerySchema.safeParse('  ').success).toBe(false);
    expect(knowledgeSearchQuerySchema.safeParse('退款').success).toBe(true);
  });

  it('uses the contract validator and maps errors to local copy', () => {
    expect(knowledgeArticleETag(3)).toBe('"knowledge-article-v3"');
    expect(() => knowledgeArticleETag(0)).toThrow(ApiError);
    expect(
      resolveKnowledgeErrorKey(new ApiError('x', ApiErrorCode.KNOWLEDGE_VERSION_CONFLICT))
    ).toBe('errors.versionConflict');
    expect(
      resolveKnowledgeErrorKey(new ApiError('x', ApiErrorCode.KNOWLEDGE_ARTICLE_NOT_FOUND))
    ).toBe('errors.notFound');
    expect(resolveKnowledgeErrorKey(new ApiError('x', ApiErrorCode.PERMISSION_DENIED))).toBe(
      'errors.forbidden'
    );
    expect(resolveKnowledgeErrorKey(new ApiError('x', ClientErrorCode.INVALID_RESPONSE))).toBe(
      'errors.invalidResponse'
    );
    expect(resolveKnowledgeErrorKey(new Error('x'))).toBe('errors.generic');
  });
});

function article() {
  return {
    id: 7,
    title: 'Refunds',
    body: '# Refunds',
    version: 2,
    created_by: null,
    updated_by: 3,
    created_at: '2026-10-11T08:00:00Z',
    updated_at: '2026-10-11T09:00:00+08:00',
  };
}
