import { declaredBodyLength, readBoundedBody } from '@/server/http/bounded-body';

type JsonBodyResult =
  | {
      ok: true;
      data: unknown;
    }
  | {
      ok: false;
      error: 'invalid' | 'too_large';
    };

const DEFAULT_MAX_JSON_BODY_BYTES = 64 * 1_024;

export async function readJsonBody(
  request: Request,
  maxBytes = DEFAULT_MAX_JSON_BODY_BYTES
): Promise<JsonBodyResult> {
  const declaredLength = declaredBodyLength(request.headers, maxBytes);
  if (declaredLength !== 'accepted') {
    return { ok: false, error: declaredLength };
  }

  const body = await readBoundedBody(request.body, maxBytes);
  if (!body.ok) {
    return body;
  }

  try {
    return {
      ok: true,
      data: JSON.parse(body.text) as unknown,
    };
  } catch {
    return {
      ok: false,
      error: 'invalid',
    };
  }
}
