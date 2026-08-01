import { z } from 'zod';

export const readinessSchema = z.object({
  status: z.enum(['degraded', 'up']),
});

export type Readiness = z.infer<typeof readinessSchema>;
