import {
  privateKnowledgeResponse,
  searchKnowledgeArticlesRoute,
} from '@/features/knowledge/server/knowledge-route';

export const dynamic = 'force-dynamic';

export async function GET(request: Request) {
  return privateKnowledgeResponse(await searchKnowledgeArticlesRoute(request));
}
