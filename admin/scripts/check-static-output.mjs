import { access, readFile, readdir, stat } from 'node:fs/promises';
import { join } from 'node:path';

const root = new URL('../dist/', import.meta.url);
const required = ['index.html', '.vite/manifest.json'];

for (const relativePath of required) {
  await access(new URL(relativePath, root));
}

const indexHtml = await readFile(new URL('index.html', root), 'utf8');
if (!indexHtml.includes('type="module"') || !indexHtml.includes('/assets/')) {
  throw new Error('dist/index.html must reference the production module assets');
}

async function walk(directory) {
  const files = [];
  for (const entry of await readdir(directory)) {
    const path = join(directory, entry);
    const details = await stat(path);
    if (details.isDirectory()) {
      files.push(...(await walk(path)));
    } else {
      files.push(path);
    }
  }
  return files;
}

const files = await walk(root.pathname);
const sourceMaps = files.filter((path) => path.endsWith('.map'));
if (sourceMaps.length > 0) {
  throw new Error(
    `Static production output must not contain source maps: ${sourceMaps.join(', ')}`,
  );
}

const serverArtifacts = files.filter((path) => /(?:^|\/)(?:server|ssr)(?:\/|\.|$)/i.test(path));
if (serverArtifacts.length > 0) {
  throw new Error(`SPA output must not contain a server bundle: ${serverArtifacts.join(', ')}`);
}

console.log(`Static output check passed (${files.length} files, no server runtime).`);
