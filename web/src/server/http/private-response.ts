import 'server-only';

const PRIVATE_NO_STORE = 'private, no-store';

export function privateNoStoreHeaders(
  initial?: HeadersInit,
  varyBy: readonly string[] = []
): Headers {
  const headers = new Headers(initial);
  headers.set('cache-control', PRIVATE_NO_STORE);
  for (const name of varyBy) {
    addVary(headers, name);
  }
  return headers;
}

export function privateNoStoreResponse<T extends Response>(
  response: T,
  varyBy: readonly string[] = []
): T {
  response.headers.set('cache-control', PRIVATE_NO_STORE);
  for (const name of varyBy) {
    addVary(response.headers, name);
  }
  return response;
}

function addVary(headers: Headers, name: string): void {
  const current = headers.get('vary');
  if (current === '*') return;

  const values = current
    ? current.split(',').map((value) => value.trim()).filter(Boolean)
    : [];
  if (values.some((value) => value.toLowerCase() === name.toLowerCase())) {
    return;
  }
  headers.set('vary', [...values, name].join(', '));
}
