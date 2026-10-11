// Registry Web item guard.
//
// Every registry plugin that ships Web TypeScript without a starter or contract fragment (themes
// and blocks) is checked before anyone installs it: its web/ files are copied into a temporary
// overlay (web/.registry-check/<plugin>/), where `@/` still resolves to this project's src/, and
// then type-checked, linted, scanned for hard-coded copy and colors, and tested with Vitest.
// Theme stylesheets are checked by check-theme-contrast.mjs. Skips cleanly when the repository
// registry is absent, as in projects created from the scaffold.
import { spawnSync } from 'node:child_process';
import {
  cpSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
  writeFileSync,
} from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const sourceRoot = join(webRoot, 'src');
const registryPlugins = resolve(webRoot, '../registry/plugins');
const overlayName = '.registry-check';
const overlayRoot = join(webRoot, overlayName);
const require = createRequire(import.meta.url);

const sourcePattern = /\.(ts|tsx)$/;
const testPattern = /\.test\.(ts|tsx)$/;
const resolvableExtensions = ['', '.ts', '.tsx', '.d.ts', '/index.ts', '/index.tsx'];
const paletteNames =
  'slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose';
const colorRules = [
  [new RegExp(`\\b[a-z-]+-(?:${paletteNames})-\\d{2,3}\\b`), 'Tailwind palette color; use a theme token'],
  [/\b(?:bg|text|border|ring|fill|stroke|outline|divide|from|via|to)-(?:white|black)\b/, 'fixed white/black; use a theme token'],
  [/#[0-9a-f]{3,8}\b/i, 'hex color; use a theme token'],
  [/\b(?:rgba?|hsla?|oklch|oklab|lab|lch)\(/, 'literal color function; use a theme token'],
  [/\btext-destructive\b/, 'use text-error for readable copy; destructive is a surface token'],
];

function listFiles(directory) {
  return readdirSync(directory).flatMap((entry) => {
    const path = join(directory, entry);
    return statSync(path).isDirectory() ? listFiles(path) : [path];
  });
}

function firstExisting(base) {
  return resolvableExtensions.map((extension) => base + extension).find((path) => existsSync(path) && statSync(path).isFile());
}

function lineOf(source, index) {
  return source.slice(0, index).split('\n').length;
}

function run(label, command, args) {
  console.log(`\n> ${label}`);
  const result = spawnSync(process.execPath, [command, ...args], { cwd: webRoot, stdio: 'inherit' });
  return result.status === 0;
}

if (!existsSync(registryPlugins)) {
  console.log('Registry item check skipped: no ../registry/plugins directory.');
  process.exit(0);
}

const failures = [];
const plugins = [];

for (const name of readdirSync(registryPlugins).sort()) {
  const manifestPath = join(registryPlugins, name, 'plugin.json');
  const webFiles = join(registryPlugins, name, 'files/web');
  if (!existsSync(manifestPath) || !existsSync(webFiles)) {
    continue;
  }
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
  if (manifest.starter || manifest.fragment) {
    continue; // feature plugins are verified by the Go install test and their own integration steps
  }
  const files = listFiles(webFiles).filter((file) => sourcePattern.test(file));
  if (files.length === 0) {
    continue;
  }
  plugins.push({ name, webFiles, files });

  for (const file of files) {
    const label = relative(resolve(webRoot, '..'), file);
    const source = readFileSync(file, 'utf8');

    for (const match of source.matchAll(/(?:from\s+|import\s*\(\s*|vi\.mock\(\s*)['"]([^'"]+)['"]/g)) {
      const specifier = match[1];
      const line = lineOf(source, match.index);
      if (specifier.startsWith('@/')) {
        const own = firstExisting(join(webFiles, 'src', specifier.slice(2)));
        if (own) {
          failures.push(`${label}:${line}: import the plugin's own file "${specifier}" with a relative path`);
        } else if (!firstExisting(join(sourceRoot, specifier.slice(2)))) {
          failures.push(`${label}:${line}: "${specifier}" does not exist in web/src`);
        }
      } else if (specifier.startsWith('.')) {
        const target = firstExisting(resolve(dirname(file), specifier));
        if (!target || !target.startsWith(webFiles + '/')) {
          failures.push(`${label}:${line}: relative import "${specifier}" must resolve inside the plugin`);
        }
      }
    }

    if (!testPattern.test(file)) {
      for (const [pattern, message] of colorRules) {
        const global = new RegExp(pattern.source, pattern.flags.includes('g') ? pattern.flags : `${pattern.flags}g`);
        for (const match of source.matchAll(global)) {
          failures.push(`${label}:${lineOf(source, match.index)}: "${match[0]}" — ${message}`);
        }
      }
      if (/\b(?:fetch|axios|useQuery|useMutation)\s*\(/.test(source)) {
        failures.push(`${label}: blocks receive data through props; do not fetch`);
      }
    }
  }
}

if (plugins.length === 0) {
  console.log('Registry item check passed: no registry plugins ship Web TypeScript without a starter.');
  process.exit(0);
}

if (failures.length > 0) {
  console.error(`Registry item check failed:\n${failures.map((failure) => `- ${failure}`).join('\n')}`);
  process.exit(1);
}

rmSync(overlayRoot, { recursive: true, force: true });
let passed = false;
try {
  for (const plugin of plugins) {
    cpSync(plugin.webFiles, join(overlayRoot, plugin.name), { recursive: true });
  }
  // The overlay extends the project config, so `@/` keeps resolving to web/src; the setup file
  // brings in the Testing Library matcher types the block tests use.
  mkdirSync(overlayRoot, { recursive: true });
  writeFileSync(
    join(overlayRoot, 'tsconfig.json'),
    `${JSON.stringify(
      {
        extends: '../tsconfig.json',
        compilerOptions: { incremental: false, plugins: [], types: ['node'] },
        include: ['../src/test/setup.ts', '**/*.ts', '**/*.tsx'],
        exclude: [],
      },
      null,
      2
    )}\n`
  );

  const fileCount = plugins.reduce((count, plugin) => count + plugin.files.length, 0);
  console.log(
    `Checking ${plugins.length} registry Web item(s), ${fileCount} TypeScript file(s): ${plugins
      .map((plugin) => plugin.name)
      .join(', ')}`
  );

  const typeChecked = run('TypeScript', require.resolve('typescript/bin/tsc'), [
    '-p',
    join(overlayName, 'tsconfig.json'),
    '--pretty',
    'false',
  ]);
  const linted = run('ESLint', join(dirname(require.resolve('eslint/package.json')), 'bin/eslint.js'), [
    '--no-ignore',
    '--max-warnings=0',
    overlayName,
  ]);
  const copyChecked = run('i18n copy', join(webRoot, 'scripts/check-i18n-copy.mjs'), [overlayName]);
  const tested = run('Vitest', join(dirname(require.resolve('vitest/package.json')), 'vitest.mjs'), [
    'run',
    '--dir',
    overlayName,
    '--passWithNoTests',
  ]);
  passed = typeChecked && linted && copyChecked && tested;
} finally {
  rmSync(overlayRoot, { recursive: true, force: true });
}

if (!passed) {
  console.error('\nRegistry item check failed; see the output above.');
  process.exit(1);
}
console.log(`\nRegistry item check passed: ${plugins.map((plugin) => plugin.name).join(', ')}.`);
