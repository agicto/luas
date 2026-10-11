import {
  array,
  boolean,
  int,
  iso,
  literal,
  maximum,
  minimum,
  nullable,
  number,
  optional,
  positive,
  refine,
  regex,
  strictObject,
  string,
} from 'zod/mini';

/** Limits from `contracts/KNOWLEDGE.md`; lengths count Unicode characters, as the API does. */
export const KNOWLEDGE_TITLE_MAX_LENGTH = 200;
export const KNOWLEDGE_BODY_MAX_LENGTH = 200_000;
export const KNOWLEDGE_QUERY_MAX_LENGTH = 200;
export const KNOWLEDGE_EXCERPT_MAX_LENGTH = 240;
export const KNOWLEDGE_SEARCH_MAX_RESULTS = 20;
export const KNOWLEDGE_PAGE_MAX_SIZE = 100;

const safeIdSchema = number().check(int(), positive(), maximum(Number.MAX_SAFE_INTEGER));
const versionSchema = safeIdSchema;
const timestampSchema = iso.datetime({ offset: true });
const actorSchema = nullable(safeIdSchema);
const storedTitleSchema = string().check(
  refine(value => characterCount(value) >= 1 && characterCount(value) <= KNOWLEDGE_TITLE_MAX_LENGTH)
);
const storedBodySchema = string().check(
  refine(value => characterCount(value) <= KNOWLEDGE_BODY_MAX_LENGTH)
);

export const knowledgeArticleSchema = strictObject({
  id: safeIdSchema,
  title: storedTitleSchema,
  body: storedBodySchema,
  version: versionSchema,
  created_by: actorSchema,
  updated_by: actorSchema,
  created_at: timestampSchema,
  updated_at: timestampSchema,
});

export const knowledgeArticleSummarySchema = strictObject({
  id: safeIdSchema,
  title: storedTitleSchema,
  version: versionSchema,
  updated_at: timestampSchema,
});

export const knowledgeSearchHitSchema = strictObject({
  id: safeIdSchema,
  title: storedTitleSchema,
  excerpt: string().check(refine(value => characterCount(value) <= KNOWLEDGE_EXCERPT_MAX_LENGTH)),
  version: versionSchema,
  updated_at: timestampSchema,
});

export const knowledgeCursorPageMetaSchema = strictObject({
  per_page: number().check(int(), minimum(1), maximum(KNOWLEDGE_PAGE_MAX_SIZE)),
  has_more: boolean(),
  next_cursor: nullable(string().check(refine(value => value.length >= 1 && value.length <= 128))),
});

export const knowledgeArticlePageEnvelopeSchema = strictObject({
  code: literal(0),
  message: string(),
  data: array(knowledgeArticleSummarySchema),
  meta: knowledgeCursorPageMetaSchema,
});

export const knowledgeSearchResultSchema = array(knowledgeSearchHitSchema).check(
  refine(values => values.length <= KNOWLEDGE_SEARCH_MAX_RESULTS)
);

/** A title the API accepts: 1-200 characters that are not all whitespace. */
const inputTitleSchema = string().check(
  refine(
    value =>
      value.trim().length > 0 &&
      characterCount(value) <= KNOWLEDGE_TITLE_MAX_LENGTH &&
      !containsControl(value)
  )
);
const inputBodySchema = string().check(
  refine(value => characterCount(value) <= KNOWLEDGE_BODY_MAX_LENGTH)
);

export const createKnowledgeArticleSchema = strictObject({
  title: inputTitleSchema,
  body: optional(inputBodySchema),
});

export const updateKnowledgeArticleSchema = strictObject({
  title: optional(inputTitleSchema),
  body: optional(inputBodySchema),
}).check(refine(value => value.title !== undefined || value.body !== undefined));

export const knowledgeSearchQuerySchema = string().check(
  refine(value => value.trim().length > 0 && characterCount(value) <= KNOWLEDGE_QUERY_MAX_LENGTH)
);

/** The opaque cursor the API returns; the client only bounds it before echoing it back. */
export const knowledgeCursorSchema = string().check(regex(/^[A-Za-z0-9_-]{1,32}$/u));

export const knowledgeRouteIdSchema = string().check(regex(/^[1-9]\d{0,15}$/u));

/** Counts Unicode code points, so a character outside the BMP counts once, as in Go. */
export function characterCount(value: string): number {
  let count = 0;
  for (let index = 0; index < value.length; index += 1) {
    const unit = value.charCodeAt(index);
    if (unit >= 0xd800 && unit <= 0xdbff && index + 1 < value.length) {
      const next = value.charCodeAt(index + 1);
      if (next >= 0xdc00 && next <= 0xdfff) index += 1;
    }
    count += 1;
  }
  return count;
}

function containsControl(value: string): boolean {
  return /[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F]/u.test(value);
}
