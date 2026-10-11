import type { z } from 'zod/mini';

import {
  assertContract,
  type Accepts,
  type ContractData,
  type ContractRequest,
  type ContractResponse,
  type Sends,
} from '@/http/contract';
import { ClientErrorCode } from '@/http/codes';
import request, { ApiError } from '@/http/request';
import {
  KNOWLEDGE_PAGE_MAX_SIZE,
  knowledgeArticlePageEnvelopeSchema,
  knowledgeArticleSchema,
  knowledgeCursorSchema,
  knowledgeSearchQuerySchema,
  knowledgeSearchResultSchema,
} from '@/features/knowledge/schemas';
import type {
  CreateKnowledgeArticleInput,
  KnowledgeArticle,
  KnowledgeArticlePage,
  KnowledgeSearchHit,
  UpdateKnowledgeArticleInput,
} from '@/features/knowledge/types';

interface ListOptions {
  cursor?: string | null;
  perPage?: number;
}

/**
 * Browser service for the knowledge routes in `contracts/KNOWLEDGE.md`. Every call carries the
 * active organization; writes carry the version that was read as `If-Match`.
 */
export const knowledgeService = {
  async list(
    organizationId: number,
    { cursor = null, perPage = 25 }: ListOptions = {}
  ): Promise<KnowledgeArticlePage> {
    if (!Number.isSafeInteger(perPage) || perPage < 1 || perPage > KNOWLEDGE_PAGE_MAX_SIZE) {
      throw invalidResponse();
    }
    if (cursor !== null && !knowledgeCursorSchema.safeParse(cursor).success) {
      throw invalidResponse();
    }
    const value = await request.getEnvelope<unknown>('/knowledge-articles', {
      headers: organizationHeaders(organizationId),
      params: { per_page: perPage, ...(cursor === null ? {} : { cursor }) },
    });
    const parsed = knowledgeArticlePageEnvelopeSchema.safeParse(value);
    if (!parsed.success) throw invalidResponse();
    if (parsed.data.meta.has_more !== (parsed.data.meta.next_cursor !== null)) {
      throw invalidResponse();
    }
    return { items: parsed.data.data, meta: parsed.data.meta };
  },

  async get(organizationId: number, articleId: number): Promise<KnowledgeArticle> {
    validId(articleId);
    const value = await request.get<unknown>(`/knowledge-articles/${articleId}`, {
      headers: organizationHeaders(organizationId),
    });
    return parseArticle(value, articleId);
  },

  async search(organizationId: number, query: string): Promise<KnowledgeSearchHit[]> {
    if (!knowledgeSearchQuerySchema.safeParse(query).success) throw invalidResponse();
    const value = await request.get<unknown>('/knowledge-search', {
      headers: organizationHeaders(organizationId),
      params: { q: query },
    });
    const parsed = knowledgeSearchResultSchema.safeParse(value);
    if (!parsed.success) throw invalidResponse();
    return parsed.data;
  },

  async create(
    organizationId: number,
    input: CreateKnowledgeArticleInput
  ): Promise<KnowledgeArticle> {
    const value = await request.post<unknown, CreateKnowledgeArticleInput>(
      '/knowledge-articles',
      input,
      { headers: organizationHeaders(organizationId) }
    );
    return parseArticle(value);
  },

  async update(
    organizationId: number,
    articleId: number,
    input: UpdateKnowledgeArticleInput,
    expectedVersion: number
  ): Promise<KnowledgeArticle> {
    validId(articleId);
    const value = await request.patch<unknown, UpdateKnowledgeArticleInput>(
      `/knowledge-articles/${articleId}`,
      input,
      { headers: mutationHeaders(organizationId, expectedVersion) }
    );
    return parseArticle(value, articleId);
  },

  async remove(organizationId: number, articleId: number, expectedVersion: number): Promise<void> {
    validId(articleId);
    await request.delete<void>(`/knowledge-articles/${articleId}`, {
      headers: mutationHeaders(organizationId, expectedVersion),
    });
  },
};

export function parseArticle(value: unknown, expectedId?: number): KnowledgeArticle {
  const parsed = knowledgeArticleSchema.safeParse(value);
  if (!parsed.success || (expectedId !== undefined && parsed.data.id !== expectedId)) {
    throw invalidResponse();
  }
  return parsed.data;
}

/** The validator the API issues as `ETag` and expects back as `If-Match`. */
export function knowledgeArticleETag(version: number): string {
  validId(version);
  return `"knowledge-article-v${version}"`;
}

function organizationHeaders(organizationId: number): Record<string, string> {
  validId(organizationId);
  return { 'Organization-Id': String(organizationId) };
}

function mutationHeaders(organizationId: number, version: number): Record<string, string> {
  return { ...organizationHeaders(organizationId), 'If-Match': knowledgeArticleETag(version) };
}

function validId(value: number): void {
  if (!Number.isSafeInteger(value) || value < 1) throw invalidResponse();
}

function invalidResponse(): ApiError {
  return new ApiError(
    'Knowledge service returned an invalid response',
    ClientErrorCode.INVALID_RESPONSE
  );
}

assertContract<
  Accepts<
    z.input<typeof knowledgeArticlePageEnvelopeSchema>,
    ContractResponse<'listKnowledgeArticles', 200>
  >
>();
assertContract<
  Accepts<z.input<typeof knowledgeArticleSchema>, ContractData<'getKnowledgeArticle', 200>>
>();
assertContract<
  Accepts<z.input<typeof knowledgeArticleSchema>, ContractData<'createKnowledgeArticle', 201>>
>();
assertContract<Sends<CreateKnowledgeArticleInput, ContractRequest<'createKnowledgeArticle'>>>();
assertContract<
  Accepts<z.input<typeof knowledgeArticleSchema>, ContractData<'updateKnowledgeArticle', 200>>
>();
assertContract<Sends<UpdateKnowledgeArticleInput, ContractRequest<'updateKnowledgeArticle'>>>();
assertContract<
  Accepts<z.input<typeof knowledgeSearchResultSchema>, ContractData<'searchKnowledgeArticles', 200>>
>();
