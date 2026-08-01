import { http } from '@/http/client';
import { readinessSchema, type Readiness } from '@/features/system/types';

export const systemService = {
  readiness(signal?: AbortSignal): Promise<Readiness> {
    return http.get('/health/ready', {
      responseMode: 'json',
      schema: readinessSchema,
      ...(signal ? { signal } : {}),
    });
  },
};
