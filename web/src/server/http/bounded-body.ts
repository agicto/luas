import 'server-only';

export type BoundedBodyResult =
  | { ok: true; text: string }
  | { ok: false; error: 'invalid' | 'too_large' };

export function declaredBodyLength(
  headers: Headers,
  maxBytes: number
): 'accepted' | 'invalid' | 'too_large' {
  const value = headers.get('content-length');
  if (value === null) {
    return 'accepted';
  }
  if (!/^(0|[1-9]\d*)$/.test(value)) {
    return 'invalid';
  }

  const length = Number(value);
  if (!Number.isSafeInteger(length)) {
    return 'too_large';
  }

  return length > maxBytes ? 'too_large' : 'accepted';
}

export async function readBoundedBody(
  body: ReadableStream<Uint8Array> | null,
  maxBytes: number
): Promise<BoundedBodyResult> {
  if (!Number.isSafeInteger(maxBytes) || maxBytes < 1 || body === null) {
    return { ok: false, error: 'invalid' };
  }

  const reader = body.getReader();
  const chunks: Uint8Array[] = [];
  let length = 0;

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      if (!(value instanceof Uint8Array)) {
        await reader.cancel();
        return { ok: false, error: 'invalid' };
      }

      length += value.byteLength;
      if (length > maxBytes) {
        await reader.cancel();
        return { ok: false, error: 'too_large' };
      }
      chunks.push(value);
    }
  } catch {
    return { ok: false, error: 'invalid' };
  } finally {
    reader.releaseLock();
  }

  const bytes = new Uint8Array(length);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }

  try {
    return {
      ok: true,
      text: new TextDecoder('utf-8', { fatal: true }).decode(bytes),
    };
  } catch {
    return { ok: false, error: 'invalid' };
  }
}
