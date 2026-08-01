import { readdir, readFile, stat } from 'node:fs/promises';
import { join } from 'node:path';
import { gzipSync } from 'node:zlib';

const assetsDirectory = new URL('../dist/assets/', import.meta.url).pathname;
const limits = {
  totalJavaScriptGzip: 300 * 1024,
  largestJavaScriptChunkGzip: 180 * 1024,
  totalCssGzip: 50 * 1024,
};

async function filesWithin(directory) {
  const files = [];
  for (const entry of await readdir(directory)) {
    const path = join(directory, entry);
    const details = await stat(path);
    if (details.isDirectory()) {
      files.push(...(await filesWithin(path)));
    } else {
      files.push(path);
    }
  }
  return files;
}

async function compressedBytes(path) {
  return gzipSync(await readFile(path), { level: 9 }).byteLength;
}

const assets = await filesWithin(assetsDirectory);
const javascript = assets.filter((path) => path.endsWith('.js'));
const stylesheets = assets.filter((path) => path.endsWith('.css'));
const javascriptSizes = await Promise.all(javascript.map(compressedBytes));
const stylesheetSizes = await Promise.all(stylesheets.map(compressedBytes));

const totalJavaScriptGzip = javascriptSizes.reduce((total, size) => total + size, 0);
const largestJavaScriptChunkGzip = Math.max(0, ...javascriptSizes);
const totalCssGzip = stylesheetSizes.reduce((total, size) => total + size, 0);

const measurements = {
  totalJavaScriptGzip,
  largestJavaScriptChunkGzip,
  totalCssGzip,
};
const failures = Object.entries(measurements).filter(([name, value]) => value > limits[name]);

if (failures.length > 0) {
  for (const [name, value] of failures) {
    console.error(`${name}: ${value} bytes exceeds ${limits[name]} bytes`);
  }
  process.exit(1);
}

console.log(
  [
    `SPA bundle budget passed (${javascript.length} JS chunks, ${stylesheets.length} CSS chunks)`,
    `JS total gzip ${totalJavaScriptGzip} B`,
    `largest JS gzip ${largestJavaScriptChunkGzip} B`,
    `CSS total gzip ${totalCssGzip} B`,
  ].join('; '),
);
