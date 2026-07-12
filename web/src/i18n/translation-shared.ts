import { AVAILABLE_MODULES, type ModuleName } from './module-names';
import type { MessageSchema } from './modules';
import type { ICUVariableNames } from './locale-message-shape';

export type TranslationValue = string | number | Date;
export type TranslationValues = Record<string, TranslationValue>;
export type BaseTranslator = (key: string, values?: TranslationValues) => string;

type StringKeyOf<T> = Extract<keyof T, string>;

type TranslationLeafPaths<T> = T extends object
  ? {
      [K in StringKeyOf<T>]: T[K] extends string
        ? K
        : T[K] extends object
          ? `${K}.${TranslationLeafPaths<T[K]> & string}`
          : never;
    }[StringKeyOf<T>]
  : never;

type TranslationScopePaths<T> = T extends object
  ? {
      [K in StringKeyOf<T>]: T[K] extends object
        ? K | `${K}.${TranslationScopePaths<T[K]> & string}`
        : never;
    }[StringKeyOf<T>]
  : never;

type ValueAtPath<T, Path extends string> = Path extends `${infer Head}.${infer Tail}`
  ? Head extends keyof T
    ? ValueAtPath<T[Head], Tail>
    : never
  : Path extends keyof T
    ? T[Path]
    : never;

export type AllTranslationKeys = TranslationLeafPaths<MessageSchema>;
export type AllScopePaths = TranslationScopePaths<MessageSchema>;
export type ScopedTranslationKeys<P extends AllScopePaths> = TranslationLeafPaths<
  ValueAtPath<MessageSchema, P>
>;

export type TranslationVariables<K extends AllTranslationKeys> =
  ValueAtPath<MessageSchema, K> extends infer Message extends string
    ? [ICUVariableNames<Message>] extends [never]
      ? never
      : { [Name in ICUVariableNames<Message>]: TranslationValue }
    : never;

type ExactTranslationVariables<
  K extends AllTranslationKeys,
  Actual extends TranslationVariables<K>,
> = Actual & Record<Exclude<keyof Actual, keyof TranslationVariables<K>>, never>;

type TranslationArguments<
  K extends AllTranslationKeys,
  Actual extends TranslationVariables<K> = TranslationVariables<K>,
> = [TranslationVariables<K>] extends [never] ? [] : [values: ExactTranslationVariables<K, Actual>];

type ScopedTranslationKey<P extends AllScopePaths, K extends ScopedTranslationKeys<P>> = Extract<
  `${P}.${K}`,
  AllTranslationKeys
>;

export type ScopedTranslations<P extends AllScopePaths> = <
  K extends ScopedTranslationKeys<P>,
  Actual extends TranslationVariables<ScopedTranslationKey<P, K>> = TranslationVariables<
    ScopedTranslationKey<P, K>
  >,
>(
  key: K,
  ...args: TranslationArguments<ScopedTranslationKey<P, K>, Actual>
) => string;

export type Translators = {
  [Namespace in ModuleName]: <
    K extends TranslationLeafPaths<MessageSchema[Namespace]>,
    FullKey extends AllTranslationKeys = Extract<`${Namespace}.${K}`, AllTranslationKeys>,
    Actual extends TranslationVariables<FullKey> = TranslationVariables<FullKey>,
  >(
    key: K,
    ...args: TranslationArguments<FullKey, Actual>
  ) => string;
};

export type UnifiedTranslations = {
  <K extends AllTranslationKeys, Actual extends TranslationVariables<K> = TranslationVariables<K>>(
    key: K,
    ...args: TranslationArguments<K, Actual>
  ): string;
} & Translators;

type ScopedTranslatorMap = Translators;

function withScope(scope: string, key: string): string {
  return `${scope}.${key}`;
}

function createScopedTranslatorMap(rootT: BaseTranslator): ScopedTranslatorMap {
  return Object.fromEntries(
    AVAILABLE_MODULES.map(moduleName => [
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
    return (key: string, values?: TranslationValues) => rootT(withScope(scope, key), values);
  }

  return Object.assign(
    (key: AllTranslationKeys, values?: TranslationValues) => rootT(key, values),
    createScopedTranslatorMap(rootT)
  ) as UnifiedTranslations;
}
