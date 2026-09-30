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
