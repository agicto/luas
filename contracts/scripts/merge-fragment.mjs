// Merges an OpenAPI fragment (written by `fastbuild make:module` or installed by `plugin:add`) into
// openapi.yaml and removes the fragment. The merge rules live in lib/fragment.mjs.
//
//   corepack pnpm merge-fragment fragments/blog_post.yaml
//   node scripts/merge-fragment.mjs --json <fragment.yaml>   prints what was added, for plugin:add
import { readFile, rm, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

import { mergeFragment } from './lib/fragment.mjs';

const schemaPath = fileURLToPath(new URL('../openapi.yaml', import.meta.url));
const json = process.argv.includes('--json');
const fragmentArgument = process.argv.slice(2).find(argument => argument !== '--json');
if (!fragmentArgument) {
  throw new Error('usage: merge-fragment <fragment.yaml>');
}
const fragmentPath = fileURLToPath(new URL(fragmentArgument, `file://${process.cwd()}/`));

const { merged, paths, schemas, additions } = mergeFragment(
  await readFile(schemaPath, 'utf8'),
  await readFile(fragmentPath, 'utf8'),
);
await writeFile(schemaPath, merged, 'utf8');
await rm(fragmentPath);
if (json) {
  console.log(JSON.stringify(additions));
  process.exit(0);
}
console.log(
  `Merged ${paths} paths and ${schemas} schemas into openapi.yaml and removed the fragment. ` +
    'Next: corepack pnpm generate && corepack pnpm check',
);
