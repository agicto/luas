/**
 * Preserve the source locale's object structure while allowing translated strings.
 */
export type LocaleMessageShape<T> = T extends string
  ? string
  : { [K in keyof T]: LocaleMessageShape<T[K]> };

type TrimLeft<S extends string> = S extends ` ${infer Rest}` ? TrimLeft<Rest> : S;
type TrimRight<S extends string> = S extends `${infer Rest} ` ? TrimRight<Rest> : S;
type Trim<S extends string> = TrimLeft<TrimRight<S>>;

type ICUVariableName<Token extends string> =
  Trim<Token> extends infer Value extends string
    ? Value extends '' | `#${string}`
      ? never
      : Value extends `${infer Name},${string}`
        ? Trim<Name>
        : Value
    : never;

export type ICUVariableNames<Message extends string> =
  Message extends `${string}{${infer Token}}${infer Rest}`
    ? ICUVariableName<Token> | ICUVariableNames<Rest>
    : never;

type SameVariables<Base extends string, Candidate extends string> = [
  ICUVariableNames<Base>,
] extends [ICUVariableNames<Candidate>]
  ? [ICUVariableNames<Candidate>] extends [ICUVariableNames<Base>]
    ? true
    : false
  : false;

/**
 * Confirm every translated leaf uses exactly the source locale's ICU variables.
 */
export type LocaleMessageVariableParity<Base, Candidate> = Base extends string
  ? Candidate extends string
    ? SameVariables<Base, Candidate>
    : false
  : Base extends object
    ? Candidate extends { [K in keyof Base]: unknown }
      ? false extends {
          [K in keyof Base]: LocaleMessageVariableParity<Base[K], Candidate[K]>;
        }[keyof Base]
        ? false
        : true
      : false
    : false;
