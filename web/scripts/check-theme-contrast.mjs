import { readFileSync, readdirSync } from 'node:fs';
import { resolve } from 'node:path';

import { parse, wcagContrast } from 'culori';

const themeRoot = resolve(process.cwd(), 'src/themes');
const sourceRoot = resolve(process.cwd(), 'src');
const primitives = readDeclarations('primitives.css');
const minimumTextContrast = 4.5;

const checks = [
  ['foreground', 'background'],
  ['card-foreground', 'card'],
  ['popover-foreground', 'popover'],
  ['text-main', 'bg-canvas'],
  ['text-main', 'bg-surface'],
  ['text-subtle', 'bg-canvas'],
  ['text-subtle', 'bg-surface'],
  ['text-muted', 'bg-canvas'],
  ['text-muted', 'bg-surface'],
  ['text-muted', 'bg-subtle'],
  ['muted-foreground', 'muted'],
  ['error', 'bg-canvas'],
  ['error', 'bg-surface'],
  ['error', 'bg-subtle'],
  ['success', 'bg-canvas'],
  ['success', 'bg-surface'],
  ['success', 'bg-subtle'],
  ['warning', 'bg-canvas'],
  ['warning', 'bg-surface'],
  ['warning', 'bg-subtle'],
  ['info', 'bg-canvas'],
  ['info', 'bg-surface'],
  ['info', 'bg-subtle'],
  ['destructive-foreground', 'destructive'],
];

function readDeclarations(file) {
  const source = readFileSync(resolve(themeRoot, file), 'utf8');
  const declarations = new Map();

  for (const match of source.matchAll(/--([a-z0-9-]+):\s*([^;]+);/gi)) {
    declarations.set(match[1], match[2].trim());
  }

  return declarations;
}

function resolveColor(token, declarations, seen = new Set()) {
  if (seen.has(token)) {
    throw new Error(`Circular color token: --${token}`);
  }

  const value = declarations.get(token) ?? primitives.get(token);
  if (!value) {
    throw new Error(`Unknown color token: --${token}`);
  }

  const reference = /^var\(--([a-z0-9-]+)\)$/i.exec(value)?.[1];
  if (reference) {
    return resolveColor(reference, declarations, new Set([...seen, token]));
  }

  const color = parse(value);
  if (!color) {
    throw new Error(`Unsupported color value for --${token}: ${value}`);
  }

  return color;
}

const failures = [];

for (const theme of ['light', 'dark']) {
  const declarations = readDeclarations(`${theme}.css`);

  for (const [foreground, background] of checks) {
    const ratio = wcagContrast(
      resolveColor(foreground, declarations),
      resolveColor(background, declarations)
    );

    if (ratio < minimumTextContrast) {
      failures.push(
        `${theme}: --${foreground} on --${background} is ${ratio.toFixed(2)}:1`
      );
    }
  }
}

function sourceFiles(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === 'test') {
        return [];
      }
      return sourceFiles(path);
    }
    return /\.(ts|tsx)$/.test(entry.name) ? [path] : [];
  });
}

for (const file of sourceFiles(sourceRoot)) {
  const source = readFileSync(file, 'utf8');
  for (const match of source.matchAll(/text-destructive/g)) {
    const line = source.slice(0, match.index).split('\n').length;
    failures.push(
      `${file.slice(process.cwd().length + 1)}:${line}: use text-error for readable copy; destructive is a surface/border token`
    );
  }
}

if (failures.length > 0) {
  console.error(
    `Theme contrast guard failed (minimum ${minimumTextContrast}:1):\n${failures
      .map((failure) => `- ${failure}`)
      .join('\n')}`
  );
  process.exit(1);
}

console.log(
  `Theme contrast guard passed: ${checks.length * 2} semantic color pairs meet ${minimumTextContrast}:1 and readable copy avoids the destructive surface token.`
);
