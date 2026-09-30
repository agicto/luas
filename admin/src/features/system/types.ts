import { z } from 'zod';

export const readinessSchema = z.object({
  status: z.enum(['degraded', 'up']),
});

export type Readiness = z.infer<typeof readinessSchema>;

export const operatorSystemSchema = z.object({
  version: z.string(),
  revision: z.string(),
  go_version: z.string(),
  starters: z.array(z.string()),
  database: z.enum(['ok', 'unavailable']),
});

export type OperatorSystem = z.infer<typeof operatorSystemSchema>;
