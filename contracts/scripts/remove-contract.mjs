// Removes what `plugin:add` merged from a plugin fragment: the tags, paths, parameters, schemas,
// and error codes named in the JSON argument. The rules live in lib/fragment.mjs.
//
//   node scripts/remove-contract.mjs '{"paths":["/v1/knowledge-articles"],"schemas":["KnowledgeArticle"]}'
import { readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

import { removeContract } from './lib/fragment.mjs';

const schemaPath = fileURLToPath(new URL('../openapi.yaml', import.meta.url));
if (!process.argv[2]) {
  throw new Error('usage: remove-contract <additions.json>');
}
const additions = JSON.parse(process.argv[2]);
await writeFile(schemaPath, removeContract(await readFile(schemaPath, 'utf8'), additions), 'utf8');
console.log('Removed the plugin additions from openapi.yaml. Next: corepack pnpm generate && corepack pnpm check');
