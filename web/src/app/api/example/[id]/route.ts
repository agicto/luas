import { z } from 'zod';
import {
  apiJsonBodyErrorResponse,
  apiNotFoundResponse,
  apiValidationErrorResponse,
} from '@/app/api/_shared/error-response';
import { readJsonBody } from '@/app/api/_shared/json-body';
import {
  guardMockBffRoute,
  guardSameOriginMutation,
} from '@/app/api/_shared/mock-bff';
import { apiSuccessResponse } from '@/app/api/_shared/success-response';
import { deleteExample, getExampleById, updateExample } from '@/features/example/server/mock-example-store';

const updateSchema = z.object({
  title: z.string().trim().min(1).max(120).optional(),
  description: z.string().trim().max(500).optional(),
  status: z.enum(['active', 'inactive']).optional(),
});

interface RouteContext {
  params: Promise<{
    id?: string | string[];
  }>;
}

async function resolveId(context: RouteContext): Promise<string> {
  const { id } = await context.params;
  return Array.isArray(id) ? id[0] ?? '' : id ?? '';
}

export async function GET(_request: Request, context: RouteContext) {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
  }

  const id = await resolveId(context);
  const exampleItem = getExampleById(id);

  if (!exampleItem) {
    return apiNotFoundResponse('Example item not found');
  }

  return apiSuccessResponse(exampleItem);
}

export async function PATCH(request: Request, context: RouteContext) {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
  }

  const sameOriginGuard = guardSameOriginMutation(request);

  if (sameOriginGuard) {
    return sameOriginGuard;
  }

  const id = await resolveId(context);
  const payload = await readJsonBody(request);

  if (!payload.ok) {
    return apiJsonBodyErrorResponse(payload.error);
  }

  const parsed = updateSchema.safeParse(payload.data);

  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid example update payload', parsed.error);
  }

  const updatedItem = updateExample(id, parsed.data);

  if (!updatedItem) {
    return apiNotFoundResponse('Example item not found');
  }

  return apiSuccessResponse(updatedItem);
}

export async function DELETE(request: Request, context: RouteContext) {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
  }

  const sameOriginGuard = guardSameOriginMutation(request);

  if (sameOriginGuard) {
    return sameOriginGuard;
  }

  const id = await resolveId(context);
  const didDelete = deleteExample(id);

  if (!didDelete) {
    return apiNotFoundResponse('Example item not found');
  }

  return apiSuccessResponse({
    id,
  });
}
