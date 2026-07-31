import { execFile } from 'node:child_process';
import { readFile } from 'node:fs/promises';
import { promisify } from 'node:util';
import { fileURLToPath } from 'node:url';

import { parse } from 'yaml';

const execute = promisify(execFile);
const apiDirectory = fileURLToPath(new URL('../../api/', import.meta.url));
const schemaPath = fileURLToPath(new URL('../openapi.yaml', import.meta.url));
const httpMethods = new Set(['delete', 'get', 'head', 'options', 'patch', 'post', 'put', 'trace']);

const schema = parse(await readFile(schemaPath, 'utf8'));
const declaredRoutes = [];
for (const [path, pathItem] of Object.entries(schema.paths ?? {})) {
  for (const method of Object.keys(pathItem ?? {})) {
    if (httpMethods.has(method)) declaredRoutes.push(routeKey(method, path));
  }
}
if (declaredRoutes.length === 0) throw new Error('OpenAPI contract has no HTTP operations');

const environment = {
  ...process.env,
  APP_ENV: 'development',
  AI_ENABLED: 'false',
  DB_ENABLED: 'false',
  LUAS_ENV_FILE: '',
  OPTIONAL_STARTERS: '',
};
const { stdout } = await execute(
  'go',
  ['run', './cmd/luas', 'route:list', '--format=json'],
  { cwd: apiDirectory, env: environment, maxBuffer: 4 * 1024 * 1024 },
);
const runtimeCatalog = JSON.parse(stdout);
const runtimeRoutes = new Set(runtimeCatalog.routes.map(route => routeKey(route.method, route.path)));
const missing = declaredRoutes.filter(route => !runtimeRoutes.has(route));
if (missing.length > 0) {
  throw new Error(`OpenAPI operations missing from the Go route assembly: ${missing.join(', ')}`);
}

console.log(
  `OpenAPI route check passed (${declaredRoutes.length} covered operations, ${runtimeRoutes.size} runtime routes).`,
);

function routeKey(method, path) {
  const normalizedPath = path.replace(/\{[^/{}]+\}/g, '{}').replace(/:[^/]+/g, '{}');
  return `${method.toUpperCase()} ${normalizedPath}`;
}
