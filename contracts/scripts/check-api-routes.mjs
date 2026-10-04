import { execFile } from 'node:child_process';
import { readFile } from 'node:fs/promises';
import { promisify } from 'node:util';
import { fileURLToPath } from 'node:url';

import { parse } from 'yaml';

const execute = promisify(execFile);
const apiDirectory = fileURLToPath(new URL('../../api/', import.meta.url));
const schemaPath = fileURLToPath(new URL('../openapi.yaml', import.meta.url));
const httpMethods = new Set(['delete', 'get', 'head', 'options', 'patch', 'post', 'put', 'trace']);

// Runtime routes that are not JSON API operations and stay outside the contract.
const undocumentedRoutes = new Map([
  ['GET /', 'service banner'],
  ['GET /metrics', 'Prometheus text exposition, not a JSON operation'],
]);

const schema = parse(await readFile(schemaPath, 'utf8'));
const declaredRoutes = [];
for (const [path, pathItem] of Object.entries(schema.paths ?? {})) {
  for (const method of Object.keys(pathItem ?? {})) {
    if (httpMethods.has(method)) declaredRoutes.push(routeKey(method, path));
  }
}
if (declaredRoutes.length === 0) throw new Error('OpenAPI contract has no HTTP operations');
const duplicates = declaredRoutes.filter((route, index) => declaredRoutes.indexOf(route) !== index);
if (duplicates.length > 0) {
  throw new Error(`OpenAPI declares the same normalized operation twice: ${duplicates.join(', ')}`);
}

// Assemble the catalog with every optional starter selected so each starter's routes are checked.
// Values below only satisfy configuration validation; nothing is served or stored.
const baseEnvironment = {
  ...process.env,
  APP_ENV: 'development',
  AI_ENABLED: 'false',
  DB_ENABLED: 'false',
  LUAS_ENV_FILE: '',
};
const starterCatalog = await luas(['starter:list', '--format=json'], {
  ...baseEnvironment,
  OPTIONAL_STARTERS: '',
});
const optionalStarters = starterCatalog.starters
  .filter(starter => starter.mode === 'optional')
  .map(starter => starter.name);
const runtimeCatalog = await luas(['route:list', '--format=json'], {
  ...baseEnvironment,
  OPTIONAL_STARTERS: optionalStarters.join(','),
  OPERATOR_ALLOWED_ORIGINS: 'http://127.0.0.1:4173',
  WEBHOOK_ENCRYPTION_KEY: 'route-catalog-only-webhook-key-0123456789abcdef',
  ASSET_TRANSFER_SIGNING_KEY: 'route-catalog-only-asset-signing-key-0123456789',
});

const runtimeRoutes = new Set(runtimeCatalog.routes.map(route => routeKey(route.method, route.path)));
const declared = new Set(declaredRoutes);

const missingFromRuntime = declaredRoutes.filter(route => !runtimeRoutes.has(route));
const missingFromContract = [...runtimeRoutes].filter(
  route => !declared.has(route) && !undocumentedRoutes.has(route),
);
const failures = [];
if (missingFromRuntime.length > 0) {
  failures.push(`OpenAPI operations missing from the Go route assembly: ${missingFromRuntime.join(', ')}`);
}
if (missingFromContract.length > 0) {
  failures.push(`Go routes missing from openapi.yaml: ${missingFromContract.sort().join(', ')}`);
}
if (failures.length > 0) throw new Error(failures.join('\n'));

console.log(
  `OpenAPI route check passed (${declaredRoutes.length} operations, ${runtimeRoutes.size} runtime routes ` +
    `with ${optionalStarters.length} optional starters, ${undocumentedRoutes.size} intentionally undocumented).`,
);

async function luas(args, env) {
  const { stdout } = await execute('go', ['run', './cmd/luas', ...args], {
    cwd: apiDirectory,
    env,
    maxBuffer: 4 * 1024 * 1024,
  });
  return JSON.parse(stdout);
}

function routeKey(method, path) {
  const normalizedPath = path.replace(/\{[^/{}]+\}/g, '{}').replace(/:[^/]+/g, '{}');
  return `${method.toUpperCase()} ${normalizedPath}`;
}
