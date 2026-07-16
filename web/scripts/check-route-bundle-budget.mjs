import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { gzipSync } from 'node:zlib';

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const statsPath = path.join(projectRoot, '.next/diagnostics/route-bundle-stats.json');
const budgetPath = path.join(projectRoot, 'performance-budgets.json');

function formatBytes(bytes) {
  return new Intl.NumberFormat('en-US').format(bytes);
}

function requirePositiveInteger(value, label) {
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new Error(`${label} must be a positive safe integer`);
  }
  return value;
}

function resolveProjectFile(relativePath) {
  const absolutePath = path.resolve(projectRoot, relativePath);
  const projectRelativePath = path.relative(projectRoot, absolutePath);
  if (projectRelativePath.startsWith('..') || path.isAbsolute(projectRelativePath)) {
    throw new Error(`route bundle path escapes the Web project: ${relativePath}`);
  }
  return absolutePath;
}

async function readJson(filePath, missingHint) {
  let source;
  try {
    source = await readFile(filePath, 'utf8');
  } catch (error) {
    if (error && typeof error === 'object' && 'code' in error && error.code === 'ENOENT') {
      throw new Error(`${filePath} is missing. ${missingHint}`);
    }
    throw error;
  }

  try {
    return JSON.parse(source);
  } catch (error) {
    throw new Error(`${filePath} is not valid JSON`, { cause: error });
  }
}

async function gzipRouteBytes(chunkPaths) {
  if (!Array.isArray(chunkPaths) || !chunkPaths.every(chunkPath => typeof chunkPath === 'string')) {
    throw new Error('route bundle stats contain invalid chunk paths');
  }

  let total = 0;
  for (const chunkPath of new Set(chunkPaths)) {
    const source = await readFile(resolveProjectFile(chunkPath));
    total += gzipSync(source, { level: 9 }).length;
  }
  return total;
}

async function main() {
  const [stats, budgetDocument] = await Promise.all([
    readJson(statsPath, 'Run pnpm build first.'),
    readJson(budgetPath, 'Restore the committed Web performance budget.'),
  ]);
  if (!Array.isArray(stats) || stats.length === 0) {
    throw new Error('Next.js route bundle stats must contain at least one route');
  }

  const policy = budgetDocument?.firstLoadUncompressedJavaScript;
  const maximumAnyRouteBytes = requirePositiveInteger(
    policy?.maximumAnyRouteBytes,
    'maximumAnyRouteBytes'
  );
  const routeBudgets = policy?.routeBudgetsBytes;
  if (!routeBudgets || typeof routeBudgets !== 'object' || Array.isArray(routeBudgets)) {
    throw new Error('routeBudgetsBytes must be an object');
  }

  const statsByRoute = new Map();
  for (const row of stats) {
    if (
      !row ||
      typeof row.route !== 'string' ||
      !Number.isSafeInteger(row.firstLoadUncompressedJsBytes) ||
      row.firstLoadUncompressedJsBytes < 0
    ) {
      throw new Error('Next.js route bundle stats contain an invalid row');
    }
    if (statsByRoute.has(row.route)) {
      throw new Error(`Next.js route bundle stats contain duplicate route ${row.route}`);
    }
    statsByRoute.set(row.route, row);
  }

  const failures = [];
  const results = [];
  for (const [route, configuredBudget] of Object.entries(routeBudgets)) {
    const budget = requirePositiveInteger(configuredBudget, `budget for ${route}`);
    const row = statsByRoute.get(route);
    if (!row) {
      failures.push(`${route} is missing from Next.js route bundle stats`);
      continue;
    }
    const actual = row.firstLoadUncompressedJsBytes;
    const gzip = await gzipRouteBytes(row.firstLoadChunkPaths);
    results.push({ route, actual, gzip, budget });
    if (actual > budget) {
      failures.push(`${route} uses ${formatBytes(actual)} bytes, budget ${formatBytes(budget)}`);
    }
  }

  const largestRoute = [...statsByRoute.values()].sort(
    (left, right) => right.firstLoadUncompressedJsBytes - left.firstLoadUncompressedJsBytes
  )[0];
  if (largestRoute.firstLoadUncompressedJsBytes > maximumAnyRouteBytes) {
    failures.push(
      `${largestRoute.route} is the largest route at ${formatBytes(largestRoute.firstLoadUncompressedJsBytes)} bytes, global budget ${formatBytes(maximumAnyRouteBytes)}`
    );
  }

  console.log('Route bundle budgets (first-load JavaScript):');
  for (const result of results) {
    const headroom = result.budget - result.actual;
    console.log(
      `  ${result.route.padEnd(45)} raw ${formatBytes(result.actual).padStart(9)} / gzip ${formatBytes(result.gzip).padStart(8)} / headroom ${formatBytes(headroom).padStart(7)}`
    );
  }
  console.log(
    `  ${'largest route'.padEnd(45)} ${largestRoute.route} at ${formatBytes(largestRoute.firstLoadUncompressedJsBytes)} raw bytes`
  );

  if (failures.length > 0) {
    console.error('Route bundle budget check failed:');
    for (const failure of failures) console.error(`  ${failure}`);
    process.exitCode = 1;
    return;
  }
  console.log('Route bundle budget check passed.');
}

main().catch(error => {
  console.error(`Route bundle budget check failed: ${error.message}`);
  process.exitCode = 1;
});
