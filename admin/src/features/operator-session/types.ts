import { z } from 'zod';

export const operatorSchema = z.object({
  id: z.number().int().positive(),
  username: z.string(),
  email: z.string(),
  nickname: z.string(),
});

export const operatorSessionSchema = z.object({
  operator: operatorSchema,
  csrf_token: z.string().min(16).max(128),
});

export type Operator = z.infer<typeof operatorSchema>;
export type OperatorSession = z.infer<typeof operatorSessionSchema>;

export interface SignInInput {
  identifier: string;
  password: string;
}
