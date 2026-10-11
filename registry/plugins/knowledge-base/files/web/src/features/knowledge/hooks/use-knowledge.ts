'use client';

import {
  keepPreviousData,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';

import { LOCAL_ERROR_HANDLING_META } from '@/config/query-meta';
import { knowledgeSearchQuerySchema } from '@/features/knowledge/schemas';
import { knowledgeService } from '@/features/knowledge/services/knowledge-service';
import type {
  CreateKnowledgeArticleInput,
  KnowledgeArticle,
  UpdateKnowledgeArticleInput,
} from '@/features/knowledge/types';

const PAGE_SIZE = 25;

export const knowledgeKeys = {
  all: ['knowledge'] as const,
  organization: (organizationId: number) => [...knowledgeKeys.all, organizationId] as const,
  articles: (organizationId: number) =>
    [...knowledgeKeys.organization(organizationId), 'articles'] as const,
  article: (organizationId: number, articleId: number) =>
    [...knowledgeKeys.organization(organizationId), 'article', articleId] as const,
  searches: (organizationId: number) =>
    [...knowledgeKeys.organization(organizationId), 'search'] as const,
  search: (organizationId: number, query: string) =>
    [...knowledgeKeys.searches(organizationId), query] as const,
};

/** Newest-first article summaries, one cursor page at a time. */
export function useKnowledgeArticles(organizationId: number) {
  return useInfiniteQuery({
    queryKey: knowledgeKeys.articles(organizationId),
    queryFn: ({ pageParam }) =>
      knowledgeService.list(organizationId, { cursor: pageParam, perPage: PAGE_SIZE }),
    initialPageParam: null as string | null,
    getNextPageParam: page => (page.meta.has_more ? page.meta.next_cursor : undefined),
    enabled: validId(organizationId),
    staleTime: 15_000,
    // The workspace renders these errors inline, so the global error toast stays quiet.
    meta: LOCAL_ERROR_HANDLING_META,
  });
}

export function useKnowledgeArticle(organizationId: number, articleId: number | null) {
  return useQuery({
    queryKey: knowledgeKeys.article(organizationId, articleId ?? 0),
    queryFn: () => knowledgeService.get(organizationId, articleId ?? 0),
    enabled: validId(organizationId) && articleId !== null && validId(articleId),
    staleTime: 15_000,
    // The workspace renders these errors inline, so the global error toast stays quiet.
    meta: LOCAL_ERROR_HANDLING_META,
  });
}

/** Runs only for a query the API accepts; callers debounce the input. */
export function useKnowledgeSearch(organizationId: number, query: string) {
  const enabled = validId(organizationId) && knowledgeSearchQuerySchema.safeParse(query).success;
  return useQuery({
    queryKey: knowledgeKeys.search(organizationId, query),
    queryFn: () => knowledgeService.search(organizationId, query),
    enabled,
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    // The workspace renders these errors inline, so the global error toast stays quiet.
    meta: LOCAL_ERROR_HANDLING_META,
  });
}

export function useCreateKnowledgeArticle(organizationId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateKnowledgeArticleInput) =>
      knowledgeService.create(organizationId, input),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: article => {
      queryClient.setQueryData(knowledgeKeys.article(organizationId, article.id), article);
    },
    onSettled: () => invalidateCollections(queryClient, organizationId),
  });
}

export function useUpdateKnowledgeArticle(organizationId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      articleId,
      input,
      expectedVersion,
    }: {
      articleId: number;
      input: UpdateKnowledgeArticleInput;
      expectedVersion: number;
    }) => knowledgeService.update(organizationId, articleId, input, expectedVersion),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: article => {
      queryClient.setQueryData<KnowledgeArticle>(
        knowledgeKeys.article(organizationId, article.id),
        article
      );
    },
    onSettled: (_article, _error, variables) => {
      void queryClient.invalidateQueries({
        queryKey: knowledgeKeys.article(organizationId, variables.articleId),
      });
      return invalidateCollections(queryClient, organizationId);
    },
  });
}

export function useDeleteKnowledgeArticle(organizationId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ articleId, expectedVersion }: { articleId: number; expectedVersion: number }) =>
      knowledgeService.remove(organizationId, articleId, expectedVersion),
    meta: LOCAL_ERROR_HANDLING_META,
    onSuccess: (_result, variables) => {
      queryClient.removeQueries({
        queryKey: knowledgeKeys.article(organizationId, variables.articleId),
      });
    },
    onSettled: () => invalidateCollections(queryClient, organizationId),
  });
}

function invalidateCollections(
  queryClient: ReturnType<typeof useQueryClient>,
  organizationId: number
) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: knowledgeKeys.articles(organizationId) }),
    queryClient.invalidateQueries({ queryKey: knowledgeKeys.searches(organizationId) }),
  ]);
}

function validId(value: number): boolean {
  return Number.isSafeInteger(value) && value > 0;
}
