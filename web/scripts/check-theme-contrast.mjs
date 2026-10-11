// Web theme contrast guard.
//
// Resolves the design tokens the way the browser cascades them: every stylesheet imported by
// src/themes/index.css, in order, where :root applies in both modes and .dark only in dark mode.
// Readable text/surface pairs must meet WCAG AA 4.5:1 in light and dark mode. When the repository
// registry is present, every registry theme's Web preset is also layered on the core tokens and
// checked, so a theme plugin cannot ship unreadable pairs or leak light values into dark mode.
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { parse, wcagContrast } from 'culori';

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const themeRoot = join(webRoot, 'src/themes');
const presetRoot = join(themeRoot, 'presets');
const sourceRoot = join(webRoot, 'src');
const registryPlugins = resolve(webRoot, '../registry/plugins');
const minimumTextContrast = 4.5;

const checks = [
  ['foreground', 'background'],
  ['card-foreground', 'card'],
  ['popover-foreground', 'popover'],
  ['primary-foreground', 'primary'],
  ['secondary-foreground', 'secondary'],
  ['accent-foreground', 'accent'],
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

function stripComments(source) {
  return source.replace(/\/\*[\s\S]*?\*\//g, '');
}

// Returns the stylesheet's rule blocks in source order: [{ selector, declarations }].
function readBlocks(file) {
  const source = stripComments(readFileSync(file, 'utf8'));
  const blocks = [];
  for (const match of source.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    const declarations = new Map();
    for (const declaration of match[2].matchAll(/--([a-z0-9-]+)\s*:\s*([^;]+);/gi)) {
      declarations.set(declaration[1], declaration[2].trim());
    }
    blocks.push({ selector: match[1].trim(), declarations });
  }
  return blocks;
}

// Local stylesheets imported by a stylesheet, in order.
function localImports(file) {
  const source = stripComments(readFileSync(file, 'utf8'));
  return [...source.matchAll(/@import\s+['"](\.[^'"]+)['"]/g)].map((match) =>
    resolve(dirname(file), match[1])
  );
}

// Token values for one mode after the cascade of the given stylesheets.
function cascade(files, mode) {
  const tokens = new Map();
  for (const file of files) {
    for (const { selector, declarations } of readBlocks(file)) {
      if (selector === ':root' || (mode === 'dark' && selector === '.dark')) {
        for (const [name, value] of declarations) {
          tokens.set(name, value);
        }
      }
    }
  }
  return tokens;
}

function resolveColor(token, tokens, seen = new Set()) {
  if (seen.has(token)) {
    throw new Error(`Circular color token: --${token}`);
  }

  const value = tokens.get(token);
  if (!value) {
    throw new Error(`Unknown color token: --${token}`);
  }

  const reference = /^var\(--([a-z0-9-]+)\)$/i.exec(value)?.[1];
  if (reference) {
    return resolveColor(reference, tokens, new Set([...seen, token]));
  }

  const color = parse(value);
  if (!color) {
    throw new Error(`Unsupported color value for --${token}: ${value}`);
  }
  if (color.alpha !== undefined && color.alpha < 1) {
    throw new Error(`--${token} is translucent (${value}); readable pairs need opaque colors`);
  }

  return color;
}

const failures = [];
let checked = 0;

function checkLayers(label, files) {
  for (const mode of ['light', 'dark']) {
    const tokens = cascade(files, mode);
    for (const [foreground, background] of checks) {
      checked += 1;
      try {
        const ratio = wcagContrast(
          resolveColor(foreground, tokens),
          resolveColor(background, tokens)
        );
        if (ratio < minimumTextContrast) {
          failures.push(
            `${label} ${mode}: --${foreground} on --${background} is ${ratio.toFixed(2)}:1`
          );
        }
      } catch (error) {
        failures.push(`${label} ${mode}: ${error.message}`);
      }
    }
  }
}

// A preset that overrides a token on :root must override it on .dark too when the core tokens
// give it a dark value; otherwise the preset's light value, imported later, wins in dark mode.
function checkPresetShape(label, presetFile, coreFiles) {
  const coreDark = new Set(
    coreFiles.flatMap((file) =>
      readBlocks(file)
        .filter((block) => block.selector === '.dark')
        .flatMap((block) => [...block.declarations.keys()])
    )
  );
  const blocks = readBlocks(presetFile);
  for (const block of blocks) {
    if (block.selector !== ':root' && block.selector !== '.dark') {
      failures.push(`${label}: only :root and .dark blocks are allowed, found "${block.selector}"`);
    }
  }
  const presetDark = new Set(
    blocks
      .filter((block) => block.selector === '.dark')
      .flatMap((block) => [...block.declarations.keys()])
  );
  for (const block of blocks.filter((candidate) => candidate.selector === ':root')) {
    for (const name of block.declarations.keys()) {
      if (coreDark.has(name) && !presetDark.has(name)) {
        failures.push(
          `${label}: --${name} is set on :root but not on .dark, so it leaks into dark mode`
        );
      }
    }
  }
}

const imported = localImports(join(themeRoot, 'index.css'));
const coreFiles = imported.filter((file) => !file.startsWith(presetRoot + '/'));
checkLayers('web', imported);
for (const file of imported.filter((candidate) => candidate.startsWith(presetRoot + '/'))) {
  checkPresetShape(relative(webRoot, file), file, coreFiles);
}

let registryThemes = 0;
if (existsSync(registryPlugins)) {
  for (const plugin of readdirSync(registryPlugins).sort()) {
    const presetDirectory = join(registryPlugins, plugin, 'files/web/src/themes/presets');
    if (!existsSync(presetDirectory)) {
      continue;
    }
    for (const name of readdirSync(presetDirectory).filter((file) => file.endsWith('.css')).sort()) {
      const presetFile = join(presetDirectory, name);
      const label = relative(resolve(webRoot, '..'), presetFile);
      registryThemes += 1;
      checkPresetShape(label, presetFile, coreFiles);
      checkLayers(label, [...coreFiles, presetFile]);
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
      `${relative(webRoot, file)}:${line}: use text-error for readable copy; destructive is a surface/border token`
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
  `Theme contrast guard passed: ${checked} semantic color pairs meet ${minimumTextContrast}:1 across the installed tokens and ${registryThemes} registry theme preset(s), and readable copy avoids the destructive surface token.`
);
