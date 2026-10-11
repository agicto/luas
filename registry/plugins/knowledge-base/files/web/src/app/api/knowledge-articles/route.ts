import {
  createKnowledgeArticleRoute,
  listKnowledgeArticlesRoute,
  privateKnowledgeResponse,
} from '@/features/knowledge/server/knowledge-route';

export const dynamic = 'force-dynamic';

export async function GET(request: Request) {
  return privateKnowledgeResponse(await listKnowledgeArticlesRoute(request));
}

export async function POST(request: Request) {
  return privateKnowledgeResponse(await createKnowledgeArticleRoute(request));
}
