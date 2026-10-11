// Proves every official plugin's contract fragment still merges into the current openapi.yaml and
// that the merged contract passes the same lint as the committed one.
import { execFile } from 'node:child_process';
import { copyFile, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { promisify } from 'node:util';
import { fileURLToPath } from 'node:url';

import { mergeFragment } from './lib/fragment.mjs';

const execute = promisify(execFile);
const contractsDir = fileURLToPath(new URL('..', import.meta.url));
const registryDir = fileURLToPath(new URL('../../registry/', import.meta.url));
const redocly = join(contractsDir, 'node_modules', '.bin', 'redocly');

const index = JSON.parse(await readFile(join(registryDir, 'index.json'), 'utf8'));
const source = await readFile(join(contractsDir, 'openapi.yaml'), 'utf8');
let checked = 0;
for (const plugin of index.plugins) {
  if (!plugin.fragment) continue;
  const fragment = await readFile(join(registryDir, 'plugins', plugin.name, 'files', plugin.fragment), 'utf8');
  const { merged } = mergeFragment(source, fragment);
  const scratch = await mkdtemp(join(tmpdir(), `fastbuild-${plugin.name}-`));
  try {
    await writeFile(join(scratch, 'openapi.yaml'), merged, 'utf8');
    await copyFile(join(contractsDir, 'redocly.yaml'), join(scratch, 'redocly.yaml'));
    await copyFile(join(contractsDir, '.redocly.lint-ignore.yaml'), join(scratch, '.redocly.lint-ignore.yaml'));
    try {
      await execute(redocly, ['lint', 'openapi.yaml'], { cwd: scratch });
    } catch (error) {
      throw new Error(`plugin ${plugin.name}: merged contract fails lint\n${error.stdout}${error.stderr}`);
    }
  } finally {
    await rm(scratch, { recursive: true, force: true });
  }
  checked += 1;
}
console.log(`Registry fragment check passed (${checked} plugin fragments merge and lint against openapi.yaml).`);
