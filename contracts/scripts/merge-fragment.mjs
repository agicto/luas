// Merges an OpenAPI fragment (written by `fastbuild make:module` or installed by `plugin:add`) into
// openapi.yaml and removes the fragment. The merge rules live in lib/fragment.mjs.
//
//   corepack pnpm merge-fragment fragments/blog_post.yaml
import { readFile, rm, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

import { mergeFragment } from './lib/fragment.mjs';

const schemaPath = fileURLToPath(new URL('../openapi.yaml', import.meta.url));
const fragmentArgument = process.argv[2];
if (!fragmentArgument) {
  throw new Error('usage: merge-fragment <fragment.yaml>');
}
const fragmentPath = fileURLToPath(new URL(fragmentArgument, `file://${process.cwd()}/`));

const { merged, paths, schemas } = mergeFragment(
  await readFile(schemaPath, 'utf8'),
  await readFile(fragmentPath, 'utf8'),
);
await writeFile(schemaPath, merged, 'utf8');
await rm(fragmentPath);
console.log(
  `Merged ${paths} paths and ${schemas} schemas into openapi.yaml and removed the fragment. ` +
    'Next: corepack pnpm generate && corepack pnpm check',
);
