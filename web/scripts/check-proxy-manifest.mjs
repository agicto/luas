import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

const manifestPath = resolve(process.cwd(), '.next/server/functions-config-manifest.json');
const expectedMatchers = [
  '/console/:path*',
  '/styleguide/:path*',
  '/i18n-test/:path*',
  '/login',
  '/register',
];

function fail(message) {
  console.error(`Proxy manifest check failed: ${message}`);
  process.exit(1);
}

let manifest;
try {
  manifest = JSON.parse(await readFile(manifestPath, 'utf8'));
} catch (error) {
  fail(`cannot read ${manifestPath}: ${error instanceof Error ? error.message : String(error)}`);
}

const proxy = manifest?.functions?.['/_middleware'];
if (!proxy || typeof proxy !== 'object') {
  fail('Next.js did not assemble src/proxy.ts as /_middleware');
}
if (proxy.runtime !== 'nodejs') {
  fail(`unexpected Proxy runtime ${JSON.stringify(proxy.runtime)}`);
}
if (!Array.isArray(proxy.matchers)) {
  fail('Proxy matcher evidence is missing');
}

const observedMatchers = proxy.matchers.map(matcher => matcher?.originalSource);
if (JSON.stringify(observedMatchers) !== JSON.stringify(expectedMatchers)) {
  fail(
    `matcher drift: observed ${JSON.stringify(observedMatchers)}, expected ${JSON.stringify(expectedMatchers)}`
  );
}
for (const matcher of proxy.matchers) {
  if (typeof matcher?.regexp !== 'string' || matcher.regexp.length === 0) {
    fail(`matcher ${JSON.stringify(matcher?.originalSource)} has no compiled regular expression`);
  }
  try {
    new RegExp(matcher.regexp);
  } catch (error) {
    fail(
      `matcher ${JSON.stringify(matcher.originalSource)} has an invalid regular expression: ${
        error instanceof Error ? error.message : String(error)
      }`
    );
  }
}

console.log(`Proxy manifest check passed with ${observedMatchers.length} exact matchers.`);
