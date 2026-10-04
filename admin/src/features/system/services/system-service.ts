import type { z } from 'zod';
import { assertContract, type Accepts, type ContractData, type ContractResponse } from '@/http/contract';
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

// Fails type-check when contracts/openapi.yaml and these schemas disagree.
assertContract<Accepts<z.input<typeof readinessSchema>, ContractResponse<'getReadiness', 200>>>();
assertContract<Accepts<z.input<typeof operatorSystemSchema>, ContractData<'getOperatorSystemStatus', 200>>>();
