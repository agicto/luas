import { describe, expect, it } from 'vitest';

import { readJsonBody } from '@/app/api/_shared/json-body';

describe('readJsonBody', () => {
  it('parses a JSON object within the configured byte budget', async () => {
    const request = new Request('https://app.example.com/api/example', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ name: 'Luas' }),
    });

    await expect(readJsonBody(request, 64)).resolves.toEqual({
      ok: true,
      data: { name: 'Luas' },
    });
  });

  it('rejects a declared body that exceeds the budget without consuming it', async () => {
    const bodyRequest = new Request('https://app.example.com/api/example', {
      method: 'POST',
      body: JSON.stringify({ name: 'Luas' }),
    });
    // happy-dom rewrites Content-Length on Request construction, so keep the
    // production Request body while supplying the ingress-declared headers.
    const request = {
      body: bodyRequest.body,
      headers: new Headers({
        'content-length': '65',
        'content-type': 'application/json',
      }),
    } as Request;

    await expect(readJsonBody(request, 64)).resolves.toEqual({
      ok: false,
      error: 'too_large',
    });
    expect(bodyRequest.bodyUsed).toBe(false);
  });

  it('stops a streaming body once the actual byte count exceeds the budget', async () => {
    const encoder = new TextEncoder();
    let cancelled = false;
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(encoder.encode('{"name":"'));
        controller.enqueue(encoder.encode('a'.repeat(80)));
        controller.enqueue(encoder.encode('"}'));
      },
      cancel() {
        cancelled = true;
      },
    });
    const request = new Request('https://app.example.com/api/example', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body,
      duplex: 'half',
    } as RequestInit & { duplex: 'half' });

    await expect(readJsonBody(request, 32)).resolves.toEqual({
      ok: false,
      error: 'too_large',
    });
    expect(cancelled).toBe(true);
  });

  it('separates malformed JSON from oversized input', async () => {
    const request = new Request('https://app.example.com/api/example', {
      method: 'POST',
      body: '{invalid',
    });

    await expect(readJsonBody(request, 64)).resolves.toEqual({
      ok: false,
      error: 'invalid',
    });
  });
});
