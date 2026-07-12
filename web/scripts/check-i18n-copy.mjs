import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs';
import { dirname, extname, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import ts from 'typescript';

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const sourceRoot = resolve(webRoot, 'src');
const coreSurfaceRoots = [
  'app/layout.tsx',
  'app/(site)',
  'components/features/site',
  'app/(auth)',
  'features/auth/components',
  'app/(protected)/(console)',
  'components/features/console',
  'components/theme-toggle.tsx',
  'components/common',
];
const sourceExtensions = new Set(['.ts', '.tsx']);
const userFacingAttributes = new Set([
  'alt',
  'aria-label',
  'description',
  'label',
  'placeholder',
  'title',
]);
const userFacingObjectKeys = new Set([
  'description',
  'emptyMessage',
  'heading',
  'label',
  'placeholder',
  'text',
  'title',
]);
const allowedBrandLiterals = new Set(['GitHub', 'Luas', 'Luas Console']);

function listSourceFiles(path) {
  const stat = statSync(path);

  if (stat.isFile()) {
    return sourceExtensions.has(extname(path)) ? [path] : [];
  }

  return readdirSync(path).flatMap(entry => listSourceFiles(resolve(path, entry)));
}

function normalizeLiteral(value) {
  return value.trim().replace(/\s+/g, ' ');
}

function hasHumanLanguage(value) {
  return /\p{L}/u.test(value);
}

function propertyName(node, source) {
  if (ts.isIdentifier(node) || ts.isStringLiteral(node)) {
    return node.text;
  }

  return node.getText(source);
}

function isInsideCodeElement(node) {
  let current = node.parent;

  while (current) {
    if (ts.isJsxElement(current)) {
      return current.openingElement.tagName.getText() === 'code';
    }
    current = current.parent;
  }

  return false;
}

function inspectCopy(path) {
  const source = ts.createSourceFile(
    path,
    readFileSync(path, 'utf8'),
    ts.ScriptTarget.Latest,
    true,
    path.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS
  );
  const violations = [];
  let allowedBrandCount = 0;

  function report(node, kind, rawValue) {
    const value = normalizeLiteral(rawValue);

    if (!hasHumanLanguage(value) || isInsideCodeElement(node)) {
      return;
    }

    if (allowedBrandLiterals.has(value)) {
      allowedBrandCount += 1;
      return;
    }

    const position = source.getLineAndCharacterOfPosition(node.getStart(source));
    violations.push({
      file: relative(sourceRoot, path),
      kind,
      line: position.line + 1,
      value,
    });
  }

  function visit(node) {
    if (ts.isJsxText(node)) {
      report(node, 'text', node.text);
    } else if (
      ts.isJsxExpression(node) &&
      node.expression &&
      (ts.isStringLiteral(node.expression) || ts.isNoSubstitutionTemplateLiteral(node.expression))
    ) {
      report(node, 'expression', node.expression.text);
    } else if (
      ts.isJsxAttribute(node) &&
      userFacingAttributes.has(node.name.getText(source)) &&
      node.initializer &&
      ts.isStringLiteral(node.initializer)
    ) {
      report(node, 'attribute', node.initializer.text);
    } else if (
      ts.isPropertyAssignment(node) &&
      userFacingObjectKeys.has(propertyName(node.name, source)) &&
      (ts.isStringLiteral(node.initializer) || ts.isNoSubstitutionTemplateLiteral(node.initializer))
    ) {
      report(node, 'object-property', node.initializer.text);
    }

    ts.forEachChild(node, visit);
  }

  visit(source);
  return { allowedBrandCount, violations };
}

const files = [
  ...new Set(
    coreSurfaceRoots.flatMap(path => {
      const absolutePath = resolve(sourceRoot, path);
      return existsSync(absolutePath) ? listSourceFiles(absolutePath) : [];
    })
  ),
].sort();
const results = files.map(inspectCopy);
const allowedBrandCount = results.reduce((count, result) => count + result.allowedBrandCount, 0);
const violations = results.flatMap(result => result.violations);

if (violations.length > 0) {
  console.error(`Core i18n copy guard found ${violations.length} hardcoded user-facing literals:`);
  for (const violation of violations) {
    console.error(
      `- ${violation.file}:${violation.line} [${violation.kind}] ${JSON.stringify(violation.value)}`
    );
  }
  process.exitCode = 1;
} else {
  console.log(
    `Core i18n copy guard passed: ${files.length} files, 0 violations, ${allowedBrandCount} allowed brand literals.`
  );
}
