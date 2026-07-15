import { z } from 'zod';
import {
  apiJsonBodyErrorResponse,
  apiValidationErrorResponse,
} from '@/app/api/_shared/error-response';
import { readJsonBody } from '@/app/api/_shared/json-body';
import {
  guardMockBffRoute,
  guardSameOriginMutation,
} from '@/app/api/_shared/mock-bff';
import { apiSuccessResponse } from '@/app/api/_shared/success-response';
import { createExample, listExamples } from '@/features/example/server/mock-example-store';

const listQuerySchema = z.object({
  keyword: z.string().optional(),
  status: z.enum(['active', 'inactive']).optional(),
  page: z.coerce.number().int().min(1).optional(),
  pageSize: z.coerce.number().int().min(1).max(100).optional(),
});

const createSchema = z.object({
  title: z.string().trim().min(1).max(120),
  description: z.string().trim().max(500).optional(),
  status: z.enum(['active', 'inactive']).optional(),
});

export async function GET(request: Request) {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
  }

  const url = new URL(request.url);
  const parsed = listQuerySchema.safeParse(Object.fromEntries(url.searchParams.entries()));

  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid query parameters', parsed.error);
  }

  return apiSuccessResponse(listExamples(parsed.data));
}

export async function POST(request: Request) {
  const mockBffGuard = guardMockBffRoute();

  if (mockBffGuard) {
    return mockBffGuard;
  }

  const sameOriginGuard = guardSameOriginMutation(request);

  if (sameOriginGuard) {
    return sameOriginGuard;
  }

  const payload = await readJsonBody(request);

  if (!payload.ok) {
    return apiJsonBodyErrorResponse(payload.error);
  }

  const parsed = createSchema.safeParse(payload.data);

  if (!parsed.success) {
    return apiValidationErrorResponse('Invalid example payload', parsed.error);
  }

  return apiSuccessResponse(createExample(parsed.data));
}
