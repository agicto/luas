import { AVAILABLE_MODULES, type ModuleName } from './module-names';
import type { Messages } from './modules';

export type TranslationValues = Record<string, string | number | Date>;
export type BaseTranslator = (key: string, values?: TranslationValues) => string;

type StringKeyOf<T> = Extract<keyof T, string>;

type TranslationLeafPaths<T> = T extends Record<string, unknown>
  ? {
      [K in StringKeyOf<T>]: T[K] extends string
        ? K
        : T[K] extends Record<string, unknown>
          ? `${K}.${TranslationLeafPaths<T[K]> & string}`
          : never;
    }[StringKeyOf<T>]
  : never;

type TranslationScopePaths<T> = T extends Record<string, unknown>
  ? {
      [K in StringKeyOf<T>]: T[K] extends Record<string, unknown>
        ? K | `${K}.${TranslationScopePaths<T[K]> & string}`
        : never;
    }[StringKeyOf<T>]
  : never;

type ValueAtPath<T, Path extends string> =
  Path extends `${infer Head}.${infer Tail}`
    ? Head extends keyof T
      ? ValueAtPath<T[Head], Tail>
      : never
    : Path extends keyof T
      ? T[Path]
      : never;

export type AllTranslationKeys = TranslationLeafPaths<Messages>;
export type AllScopePaths = TranslationScopePaths<Messages>;
export type ScopedTranslationKeys<P extends AllScopePaths> =
  TranslationLeafPaths<ValueAtPath<Messages, P>>;

export type ScopedTranslations<P extends AllScopePaths> = (
  key: ScopedTranslationKeys<P>,
  values?: TranslationValues
) => string;

export type Translators = {
  [K in ModuleName]: (
    key: TranslationLeafPaths<Messages[K]>,
    values?: TranslationValues
  ) => string;
};

export type UnifiedTranslations = {
  (key: AllTranslationKeys, values?: TranslationValues): string;
} & Translators;

type ScopedTranslatorMap = Translators;

function withScope(scope: string, key: string): string {
  return `${scope}.${key}`;
}

function createScopedTranslatorMap(rootT: BaseTranslator): ScopedTranslatorMap {
  return Object.fromEntries(
    AVAILABLE_MODULES.map((moduleName) => [
      moduleName,
      (key: string, values?: TranslationValues) => rootT(withScope(moduleName, key), values),
    ])
  ) as ScopedTranslatorMap;
}

export function createTranslator(rootT: BaseTranslator): UnifiedTranslations;
export function createTranslator<P extends AllScopePaths>(
  rootT: BaseTranslator,
  scope: P
): ScopedTranslations<P>;
export function createTranslator(
  rootT: BaseTranslator,
  scope?: AllScopePaths
): UnifiedTranslations | ScopedTranslations<AllScopePaths> {
  if (scope) {
    return (key: string, values?: TranslationValues) =>
      rootT(withScope(scope, key), values);
  }

  return Object.assign(
    (key: AllTranslationKeys, values?: TranslationValues) => rootT(key, values),
    createScopedTranslatorMap(rootT)
  ) as UnifiedTranslations;
}
