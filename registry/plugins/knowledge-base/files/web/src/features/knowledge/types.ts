import type { infer as Infer } from 'zod/mini';

import type {
  createKnowledgeArticleSchema,
  knowledgeArticleSchema,
  knowledgeArticleSummarySchema,
  knowledgeCursorPageMetaSchema,
  knowledgeSearchHitSchema,
  updateKnowledgeArticleSchema,
} from './schemas';

export type KnowledgeArticle = Infer<typeof knowledgeArticleSchema>;
export type KnowledgeArticleSummary = Infer<typeof knowledgeArticleSummarySchema>;
export type KnowledgeSearchHit = Infer<typeof knowledgeSearchHitSchema>;
export type KnowledgeCursorPageMeta = Infer<typeof knowledgeCursorPageMetaSchema>;
export type CreateKnowledgeArticleInput = Infer<typeof createKnowledgeArticleSchema>;
export type UpdateKnowledgeArticleInput = Infer<typeof updateKnowledgeArticleSchema>;

export interface KnowledgeArticlePage {
  items: KnowledgeArticleSummary[];
  meta: KnowledgeCursorPageMeta;
}
