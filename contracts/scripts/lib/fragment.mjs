// Merges an OpenAPI fragment into the contract text, and removes what a merge added. Tags, paths, parameters, schemas, and error
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
  const additions = {
    tags: sections.tags.map(tag => tag.name),
    paths: Object.keys(sections.paths),
    parameters: Object.keys(sections.parameters),
    schemas: Object.keys(sections.schemas),
    error_codes: sections.errorCodes,
  };
  return { merged, paths: additions.paths.length, schemas: additions.schemas.length, additions };
}

// Removes the tags, paths, parameters, schemas, and error codes a merge added, as text, so the
// rest of the file keeps its formatting. Names that are already gone are skipped.
export function removeContract(source, additions) {
  const lines = source.split('\n');
  // Deletes each entry of a section, at exactly `indent` spaces, whose key matches; an entry ends
  // at the next line indented at most as far. Blank lines between entries are kept.
  const removeBlock = (headerPattern, indent, matches) => {
    const header = lines.findIndex(line => headerPattern.test(line));
    if (header < 0) return;
    for (let index = header + 1; index < lines.length; index += 1) {
      const line = lines[index];
      if (line === '') continue;
      if (indentation(line) < indent) break;
      if (indentation(line) !== indent || !matches(line.slice(indent))) continue;
      let end = index + 1;
      while (end < lines.length && (lines[end] === '' || indentation(lines[end]) > indent)) end += 1;
      while (end > index + 1 && lines[end - 1] === '') end -= 1;
      lines.splice(index, end - index);
      index -= 1;
    }
  };
  const keyOf = text => text.replace(/:$/, '').replace(/^'(.*)'$/, '$1');
  const paths = new Set(additions.paths ?? []);
  const parameters = new Set(additions.parameters ?? []);
  const schemas = new Set(additions.schemas ?? []);
  const tags = new Set(additions.tags ?? []);
  const codes = new Set(additions.error_codes ?? []);
  removeBlock(/^tags:$/, 2, text => text.startsWith('- name: ') && tags.has(text.slice(8).replace(/^'(.*)'$/, '$1')));
  removeBlock(/^paths:$/, 2, text => text.endsWith(':') && paths.has(keyOf(text)));
  removeBlock(/^ {2}parameters:$/, 4, text => text.endsWith(':') && parameters.has(keyOf(text)));
  removeBlock(/^ {2}schemas:$/, 4, text => text.endsWith(':') && schemas.has(keyOf(text)));
  const start = lines.indexOf('    ErrorCode:');
  if (start >= 0) {
    for (let index = lines.indexOf('      enum:', start) + 1; lines[index]?.startsWith('        - '); index += 1) {
      if (codes.has(lines[index].slice(10))) {
        lines.splice(index, 1);
        index -= 1;
      }
    }
  }
  const removed = lines.join('\n');

  // The result must parse to exactly the contract without the additions.
  const before = parse(source);
  const after = parse(removed);
  const expect = (section, names, kept) => {
    for (const name of names) {
      if (kept(after, name)) throw new Error(`Removal left ${section} ${name}`);
    }
  };
  expect('path', paths, (doc, name) => Boolean(doc.paths?.[name]));
  expect('parameter', parameters, (doc, name) => Boolean(doc.components?.parameters?.[name]));
  expect('schema', schemas, (doc, name) => Boolean(doc.components?.schemas?.[name]));
  const countKept = section => Object.keys(section ?? {}).length;
  const removedPaths = Object.keys(before.paths ?? {}).filter(name => paths.has(name)).length;
  if (countKept(after.paths) !== countKept(before.paths) - removedPaths) {
    throw new Error('Removal changed other paths; openapi.yaml layout is not the expected one');
  }
  const removedSchemas = Object.keys(before.components?.schemas ?? {}).filter(name => schemas.has(name)).length;
  if (countKept(after.components?.schemas) !== countKept(before.components?.schemas) - removedSchemas) {
    throw new Error('Removal changed other schemas; openapi.yaml layout is not the expected one');
  }
  return removed;
}

function indentation(line) {
  return line.length - line.trimStart().length;
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
