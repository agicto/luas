import type { z } from 'zod';
import { assertContract, type Accepts, type ContractData, type ContractRequest, type Sends } from '@/http/contract';
import { http } from '@/http/client';
import {
  operatorSessionSchema,
  type OperatorSession,
  type SignInInput,
} from '@/features/operator-session/types';

export const operatorSessionService = {
  current(signal?: AbortSignal): Promise<OperatorSession> {
    return http.get('/v1/operator/session', {
      schema: operatorSessionSchema,
      ...(signal ? { signal } : {}),
    });
  },
  signIn(input: SignInInput): Promise<OperatorSession> {
    return http.post('/v1/operator/session', input, { schema: operatorSessionSchema });
  },
  signOut(): Promise<void> {
    return http.delete('/v1/operator/session', { responseMode: 'json' });
  },
};

// Fails type-check when contracts/openapi.yaml and these schemas disagree.
assertContract<Accepts<z.input<typeof operatorSessionSchema>, ContractData<'getOperatorSession', 200>>>();
assertContract<Accepts<z.input<typeof operatorSessionSchema>, ContractData<'signInOperator', 200>>>();
assertContract<Sends<SignInInput, ContractRequest<'signInOperator'>>>();
