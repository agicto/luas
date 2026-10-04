// Merges an OpenAPI fragment (written by `luas make:module`) into openapi.yaml. It inserts the
// fragment's tags, paths, parameters, and schemas as text at the end of each section, so the rest of
// the file keeps its formatting, and refuses any name that already exists.
//
//   corepack pnpm merge-fragment fragments/blog_post.yaml
import { readFile, rm, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

import { parse, stringify } from 'yaml';

const schemaPath = fileURLToPath(new URL('../openapi.yaml', import.meta.url));
const fragmentArgument = process.argv[2];
if (!fragmentArgument) {
  throw new Error('usage: merge-fragment <fragment.yaml>');
}
const fragmentPath = fileURLToPath(new URL(fragmentArgument, `file://${process.cwd()}/`));

const source = await readFile(schemaPath, 'utf8');
const schema = parse(source);
const fragment = parse(await readFile(fragmentPath, 'utf8')) ?? {};

const sections = {
  tags: fragment.tags ?? [],
  paths: fragment.paths ?? {},
  parameters: fragment.components?.parameters ?? {},
  schemas: fragment.components?.schemas ?? {},
};
const unsupported = Object.keys(fragment).filter(key => !['tags', 'paths', 'components'].includes(key));
unsupported.push(
  ...Object.keys(fragment.components ?? {})
    .filter(key => !['parameters', 'schemas'].includes(key))
    .map(key => `components.${key}`),
);
if (unsupported.length > 0) {
  throw new Error(`Fragment sections are not supported: ${unsupported.join(', ')}`);
}

const collisions = [];
const existingTags = new Set((schema.tags ?? []).map(tag => tag.name));
for (const tag of sections.tags) {
  if (existingTags.has(tag.name)) collisions.push(`tag ${tag.name}`);
}
for (const path of Object.keys(sections.paths)) {
  if (schema.paths?.[path]) collisions.push(`path ${path}`);
}
for (const name of Object.keys(sections.parameters)) {
  if (schema.components?.parameters?.[name]) collisions.push(`parameter ${name}`);
}
for (const name of Object.keys(sections.schemas)) {
  if (schema.components?.schemas?.[name]) collisions.push(`schema ${name}`);
}
const existingOperations = new Set(operationIds(schema.paths ?? {}));
for (const operationId of operationIds(sections.paths)) {
  if (existingOperations.has(operationId)) collisions.push(`operationId ${operationId}`);
}
if (collisions.length > 0) {
  throw new Error(`Fragment collides with openapi.yaml: ${collisions.join(', ')}`);
}

let merged = source.endsWith('\n') ? source : `${source}\n`;
merged = insertBefore(merged, /^paths:$/m, block(sections.tags, 2));
merged = insertBefore(merged, /^components:$/m, block(sections.paths, 2));
merged = insertBefore(merged, /^ {2}responses:$/m, block(sections.parameters, 4));
merged += block(sections.schemas, 4);

// The merged text must parse to exactly the union of both documents.
const result = parse(merged);
for (const path of Object.keys(sections.paths)) {
  if (!result.paths?.[path]) throw new Error(`Merge lost path ${path}`);
}
for (const name of Object.keys(sections.schemas)) {
  if (!result.components?.schemas?.[name]) throw new Error(`Merge lost schema ${name}`);
}
if (Object.keys(result.paths).length !== Object.keys(schema.paths).length + Object.keys(sections.paths).length) {
  throw new Error('Merge changed the existing paths; openapi.yaml layout is not the expected one');
}

await writeFile(schemaPath, merged, 'utf8');
await rm(fragmentPath);
console.log(
  `Merged ${Object.keys(sections.paths).length} paths and ${Object.keys(sections.schemas).length} schemas into ` +
    'openapi.yaml and removed the fragment. Next: corepack pnpm generate && corepack pnpm check',
);

function operationIds(paths) {
  return Object.values(paths).flatMap(item =>
    Object.values(item ?? {})
      .map(operation => operation?.operationId)
      .filter(Boolean),
  );
}

function block(value, indent) {
  const empty = Array.isArray(value) ? value.length === 0 : Object.keys(value).length === 0;
  if (empty) return '';
  const pad = ' '.repeat(indent);
  return stringify(value, { singleQuote: true, lineWidth: 100 })
    .trimEnd()
    .split('\n')
    .map(line => (line ? pad + line : line))
    .join('\n')
    .concat('\n');
}

function insertBefore(text, pattern, insertion) {
  if (!insertion) return text;
  const match = pattern.exec(text);
  if (!match) throw new Error(`openapi.yaml has no ${pattern} section`);
  return text.slice(0, match.index) + insertion + text.slice(match.index);
}
