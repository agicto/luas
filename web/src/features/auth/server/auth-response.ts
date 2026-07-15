import 'server-only';

import { privateNoStoreResponse } from '@/server/http/private-response';

export function privateAuthResponse<T extends Response>(response: T): T {
  return privateNoStoreResponse(response, ['Cookie']);
}
