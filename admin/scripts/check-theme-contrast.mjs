// Admin Console theme contrast guard.
//
// Resolves the design tokens the way the browser cascades them (every stylesheet imported by
// globals.css, in order; :root applies in both modes and .dark only in dark mode) and checks that
// readable text/surface pairs meet WCAG AA 4.5:1 in light and dark mode. When the repository
// registry is present it also checks every registry theme's Admin stylesheet layered on the core
// tokens, so a theme cannot ship unreadable pairs. Dependency-free: OKLCH is converted to sRGB here.
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const adminRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const stylesRoot = join(adminRoot, 'src/styles');
const registryPlugins = resolve(adminRoot, '../registry/plugins');
const minimumTextContrast = 4.5;

// Foreground token on background token, as the console renders them.
const checks = [
  ['foreground', 'background'],
  ['card-foreground', 'card'],
  ['popover-foreground', 'popover'],
  ['primary-foreground', 'primary'],
  ['secondary-foreground', 'secondary'],
  ['accent-foreground', 'accent'],
  ['muted-foreground', 'muted'],
  ['muted-foreground', 'background'],
  ['muted-foreground', 'card'],
  ['subtle', 'background'],
  ['subtle', 'card'],
  ['destructive', 'background'],
  ['destructive', 'card'],
  ['brand-foreground', 'brand'],
  ['brand-strong', 'background'],
  ['brand-strong', 'brand-subtle'],
  ['sidebar-accent-foreground', 'sidebar-accent'],
  ['success-strong', 'success-subtle'],
  ['warning-strong', 'warning-subtle'],
  ['danger-strong', 'danger-subtle'],
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

// Local stylesheets imported by a stylesheet, in order (package imports are skipped).
function localImports(file) {
  const source = stripComments(readFileSync(file, 'utf8'));
  return [...source.matchAll(/@import\s+['"](\.[^'"]+)['"]/g)].map((match) =>
    resolve(dirname(file), match[1]),
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

function oklchToLinearSrgb(lightness, chroma, hue) {
  const radians = (hue * Math.PI) / 180;
  const a = chroma * Math.cos(radians);
  const b = chroma * Math.sin(radians);
  const l = (lightness + 0.3963377774 * a + 0.2158037573 * b) ** 3;
  const m = (lightness - 0.1055613458 * a - 0.0638541728 * b) ** 3;
  const s = (lightness - 0.0894841775 * a - 1.291485548 * b) ** 3;
  const clamp = (value) => Math.min(1, Math.max(0, value));
  return [
    clamp(4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s),
    clamp(-1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s),
    clamp(-0.0041960863 * l - 0.7034186147 * m + 1.707614701 * s),
  ];
}

function luminance(token, tokens, seen = new Set()) {
  if (seen.has(token)) {
    throw new Error(`Circular color token: --${token}`);
  }
  const value = tokens.get(token);
  if (!value) {
    throw new Error(`Unknown color token: --${token}`);
  }
  const reference = /^var\(--([a-z0-9-]+)\)$/i.exec(value)?.[1];
  if (reference) {
    return luminance(reference, tokens, new Set([...seen, token]));
  }
  const color =
    /^oklch\(\s*([\d.]+)(%?)\s+([\d.]+)\s+([\d.]+)\s*(?:\/\s*([\d.]+)(%?)\s*)?\)$/i.exec(value);
  if (!color) {
    throw new Error(`Unsupported color value for --${token}: ${value} (use opaque oklch())`);
  }
  const alpha = color[5] === undefined ? 1 : Number(color[5]) / (color[6] ? 100 : 1);
  if (alpha < 1) {
    throw new Error(`--${token} is translucent (${value}); readable pairs need opaque colors`);
  }
  const lightness = Number(color[1]) / (color[2] ? 100 : 1);
  const [r, g, b] = oklchToLinearSrgb(lightness, Number(color[3]), Number(color[4]));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(foreground, background, tokens) {
  const [lighter, darker] = [luminance(foreground, tokens), luminance(background, tokens)].sort(
    (left, right) => right - left,
  );
  return (lighter + 0.05) / (darker + 0.05);
}

const failures = [];
let checked = 0;

function checkLayers(label, files) {
  for (const mode of ['light', 'dark']) {
    const tokens = cascade(files, mode);
    for (const [foreground, background] of checks) {
      checked += 1;
      try {
        const ratio = contrast(foreground, background, tokens);
        if (ratio < minimumTextContrast) {
          failures.push(
            `${label} ${mode}: --${foreground} on --${background} is ${ratio.toFixed(2)}:1`,
          );
        }
      } catch (error) {
        failures.push(`${label} ${mode}: ${error.message}`);
      }
    }
  }
}

// A theme that overrides a token on :root must override it on .dark too when the core tokens give
// it a dark value; otherwise the theme's light value, imported later, wins in dark mode as well.
function checkDarkCoverage(label, themeFile, coreFiles) {
  const coreDark = new Set(
    coreFiles.flatMap((file) =>
      readBlocks(file)
        .filter((block) => block.selector === '.dark')
        .flatMap((block) => [...block.declarations.keys()]),
    ),
  );
  const blocks = readBlocks(themeFile);
  for (const block of blocks) {
    if (block.selector !== ':root' && block.selector !== '.dark') {
      failures.push(`${label}: only :root and .dark blocks are allowed, found "${block.selector}"`);
    }
  }
  const themeDark = new Set(
    blocks.filter((block) => block.selector === '.dark').flatMap((block) => [...block.declarations.keys()]),
  );
  for (const block of blocks.filter((candidate) => candidate.selector === ':root')) {
    for (const name of block.declarations.keys()) {
      if (coreDark.has(name) && !themeDark.has(name)) {
        failures.push(`${label}: --${name} is set on :root but not on .dark, so it leaks into dark mode`);
      }
    }
  }
}

const globals = join(stylesRoot, 'globals.css');
const imported = localImports(globals);
const coreFiles = imported.filter((file) => !file.startsWith(join(stylesRoot, 'themes') + '/'));
checkLayers('admin', imported);

let themes = 0;
if (existsSync(registryPlugins)) {
  for (const plugin of readdirSync(registryPlugins).sort()) {
    const themeDirectory = join(registryPlugins, plugin, 'files/admin/src/styles/themes');
    if (!existsSync(themeDirectory)) {
      continue;
    }
    for (const name of readdirSync(themeDirectory).filter((file) => file.endsWith('.css')).sort()) {
      const themeFile = join(themeDirectory, name);
      const label = relative(resolve(adminRoot, '..'), themeFile);
      themes += 1;
      checkDarkCoverage(label, themeFile, coreFiles);
      checkLayers(label, [...coreFiles, themeFile]);
    }
  }
}

if (failures.length > 0) {
  console.error(
    `Admin theme contrast guard failed (minimum ${minimumTextContrast}:1):\n${failures
      .map((failure) => `- ${failure}`)
      .join('\n')}`,
  );
  process.exit(1);
}

console.log(
  `Admin theme contrast guard passed: ${checked} color pairs meet ${minimumTextContrast}:1 across the installed tokens and ${themes} registry theme(s).`,
);
