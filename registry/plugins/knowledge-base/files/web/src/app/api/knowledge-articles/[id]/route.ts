import {
  deleteKnowledgeArticleRoute,
  getKnowledgeArticleRoute,
  privateKnowledgeResponse,
  updateKnowledgeArticleRoute,
} from '@/features/knowledge/server/knowledge-route';

export const dynamic = 'force-dynamic';

interface RouteContext {
  params: Promise<{ id: string }>;
}

export async function GET(request: Request, context: RouteContext) {
  const { id } = await context.params;
  return privateKnowledgeResponse(await getKnowledgeArticleRoute(request, id));
}

export async function PATCH(request: Request, context: RouteContext) {
  const { id } = await context.params;
  return privateKnowledgeResponse(await updateKnowledgeArticleRoute(request, id));
}

export async function DELETE(request: Request, context: RouteContext) {
  const { id } = await context.params;
  return privateKnowledgeResponse(await deleteKnowledgeArticleRoute(request, id));
}
