// Merges an OpenAPI fragment into the contract text. Tags, paths, parameters, schemas, and error
// codes (`x-error-codes`, appended to the ErrorCode enum) are inserted as text at the end of each
// section, so the rest of the file keeps its formatting; any name that already exists is refused.
import { parse, stringify } from 'yaml';

export function mergeFragment(source, fragmentText) {
  const schema = parse(source);
  const fragment = parse(fragmentText) ?? {};
  const sections = {
    tags: fragment.tags ?? [],
    paths: fragment.paths ?? {},
    parameters: fragment.components?.parameters ?? {},
    schemas: fragment.components?.schemas ?? {},
    errorCodes: fragment['x-error-codes'] ?? [],
  };
  const unsupported = Object.keys(fragment).filter(
    key => !['tags', 'paths', 'components', 'x-error-codes'].includes(key),
  );
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
  const existingCodes = new Set(schema.components?.schemas?.ErrorCode?.enum ?? []);
  for (const code of sections.errorCodes) {
    if (typeof code !== 'string' || !/^[A-Z][A-Z0-9_]*\.[A-Z][A-Z0-9_]*$/.test(code)) {
      collisions.push(`invalid error code ${code}`);
    } else if (existingCodes.has(code)) {
      collisions.push(`error code ${code}`);
    }
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
  merged = appendErrorCodes(merged, sections.errorCodes);

  // The merged text must parse to exactly the union of both documents.
  const result = parse(merged);
  for (const path of Object.keys(sections.paths)) {
    if (!result.paths?.[path]) throw new Error(`Merge lost path ${path}`);
  }
  for (const name of Object.keys(sections.schemas)) {
    if (!result.components?.schemas?.[name]) throw new Error(`Merge lost schema ${name}`);
  }
  for (const code of sections.errorCodes) {
    if (!result.components.schemas.ErrorCode.enum.includes(code)) throw new Error(`Merge lost error code ${code}`);
  }
  if (Object.keys(result.paths).length !== Object.keys(schema.paths).length + Object.keys(sections.paths).length) {
    throw new Error('Merge changed the existing paths; openapi.yaml layout is not the expected one');
  }
  return { merged, paths: Object.keys(sections.paths).length, schemas: Object.keys(sections.schemas).length };
}

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

// Appends codes after the last entry of the ErrorCode enum, keeping its indentation.
function appendErrorCodes(text, codes) {
  if (codes.length === 0) return text;
  const lines = text.split('\n');
  const start = lines.indexOf('    ErrorCode:');
  if (start < 0) throw new Error('openapi.yaml has no ErrorCode schema');
  const enumLine = lines.indexOf('      enum:', start);
  if (enumLine < 0) throw new Error('ErrorCode schema has no enum');
  let last = enumLine;
  while (lines[last + 1]?.startsWith('        - ')) last += 1;
  lines.splice(last + 1, 0, ...codes.map(code => `        - ${code}`));
  return lines.join('\n');
}
