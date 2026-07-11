import { AVAILABLE_MODULES, type ModuleName } from './module-names';
import type { Messages } from './modules';

export type TranslationValues = Record<string, string | number | Date>;
export type BaseTranslator = (key: string, values?: TranslationValues) => string;

type DotNotationKeys<T, Prefix extends string = ''> = T extends object
  ? {
      [K in keyof T]: K extends string
        ? T[K] extends object
          ? `${Prefix}${K}` | DotNotationKeys<T[K], `${Prefix}${K}.`>
          : `${Prefix}${K}`
        : never;
    }[keyof T]
  : never;

export type Translators = {
  [K in keyof Messages]: (
    key: DotNotationKeys<Messages[K]>,
    values?: TranslationValues
  ) => string;
};

export type AllTranslationKeys = DotNotationKeys<Messages>;
export type AllScopePaths = string;

export type UnifiedTranslations = {
  (key: AllTranslationKeys, values?: TranslationValues): string;
} & Translators;

export type ScopedTranslations = (
  key: string,
  values?: TranslationValues
) => string;

type ScopedTranslatorMap = Record<ModuleName, ScopedTranslations>;

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
export function createTranslator(rootT: BaseTranslator, scope: string): ScopedTranslations;
export function createTranslator(
  rootT: BaseTranslator,
  scope?: string
): UnifiedTranslations | ScopedTranslations {
  if (scope) {
    return (key: string, values?: TranslationValues) =>
      rootT(withScope(scope, key), values);
  }

  return Object.assign(
    (key: AllTranslationKeys, values?: TranslationValues) => rootT(key, values),
    createScopedTranslatorMap(rootT)
  ) as UnifiedTranslations;
}
