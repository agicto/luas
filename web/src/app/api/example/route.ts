import { NextResponse } from 'next/server';
import { z } from 'zod';
import { createExample, listExamples } from '@/features/example/server/mock-example-store';
import { ErrorCode } from '@/http/codes';

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
  const url = new URL(request.url);
  const parsed = listQuerySchema.safeParse(Object.fromEntries(url.searchParams.entries()));

  if (!parsed.success) {
    return NextResponse.json(
      {
        error: 'Invalid query parameters',
        code: ErrorCode.INVALID_PARAMS,
      },
      { status: 400 }
    );
  }

  return NextResponse.json({
    data: listExamples(parsed.data),
  });
}

export async function POST(request: Request) {
  const payload = await request.json().catch(() => null);
  const parsed = createSchema.safeParse(payload);

  if (!parsed.success) {
    return NextResponse.json(
      {
        error: 'Invalid example payload',
        code: ErrorCode.INVALID_PARAMS,
      },
      { status: 400 }
    );
  }

  return NextResponse.json({
    data: createExample(parsed.data),
  });
}
