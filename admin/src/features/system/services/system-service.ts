import { http } from '@/http/client';
import {
  operatorSystemSchema,
  readinessSchema,
  type OperatorSystem,
  type Readiness,
} from '@/features/system/types';

export const systemService = {
  readiness(signal?: AbortSignal): Promise<Readiness> {
    return http.get('/health/ready', {
      responseMode: 'json',
      schema: readinessSchema,
      ...(signal ? { signal } : {}),
    });
  },
  operatorSystem(signal?: AbortSignal): Promise<OperatorSystem> {
    return http.get('/v1/operator/system', {
      schema: operatorSystemSchema,
      ...(signal ? { signal } : {}),
    });
  },
};
