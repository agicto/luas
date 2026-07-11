import type { ModuleName } from './module-names';
import type { Messages } from './modules';

export function selectMessageNamespaces<
  const Namespaces extends readonly ModuleName[],
>(
  messages: Messages,
  namespaces: Namespaces
): Pick<Messages, Namespaces[number]> {
  return Object.fromEntries(
    namespaces.map((namespace) => [namespace, messages[namespace]])
  ) as Pick<Messages, Namespaces[number]>;
}
