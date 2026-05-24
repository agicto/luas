import { NextResponse } from 'next/server';
import { z } from 'zod';
import { deleteExample, getExampleById, updateExample } from '@/features/example/server/mock-example-store';
import { ErrorCode } from '@/http/codes';

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
  const id = await resolveId(context);
  const exampleItem = getExampleById(id);

  if (!exampleItem) {
    return NextResponse.json(
      {
        error: 'Example item not found',
        code: ErrorCode.RESOURCE_NOT_FOUND,
      },
      { status: 404 }
    );
  }

  return NextResponse.json({
    data: exampleItem,
  });
}

export async function PATCH(request: Request, context: RouteContext) {
  const id = await resolveId(context);
  const payload = await request.json().catch(() => null);
  const parsed = updateSchema.safeParse(payload);

  if (!parsed.success) {
    return NextResponse.json(
      {
        error: 'Invalid example update payload',
        code: ErrorCode.INVALID_PARAMS,
      },
      { status: 400 }
    );
  }

  const updatedItem = updateExample(id, parsed.data);

  if (!updatedItem) {
    return NextResponse.json(
      {
        error: 'Example item not found',
        code: ErrorCode.RESOURCE_NOT_FOUND,
      },
      { status: 404 }
    );
  }

  return NextResponse.json({
    data: updatedItem,
  });
}

export async function DELETE(_request: Request, context: RouteContext) {
  const id = await resolveId(context);
  const didDelete = deleteExample(id);

  if (!didDelete) {
    return NextResponse.json(
      {
        error: 'Example item not found',
        code: ErrorCode.RESOURCE_NOT_FOUND,
      },
      { status: 404 }
    );
  }

  return NextResponse.json({
    data: {
      id,
    },
  });
}
