import { z } from 'zod';

export const pageMetaSchema = z.object({
  current_page: z.number().int().min(1),
  last_page: z.number().int().min(1),
  per_page: z.number().int().min(1),
  total: z.number().int().min(0),
});

export type PageMeta = z.infer<typeof pageMetaSchema>;

/** The shared paginated envelope around one item schema. */
export function pageSchema<Item extends z.ZodType>(item: Item) {
  return z.object({
    code: z.literal(0),
    data: z.array(item),
    meta: pageMetaSchema,
  });
}
