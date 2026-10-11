import { OPTIONAL_WEB_FEATURES } from '@/config/optional-features';

/**
 * Plugin Web features for the guard tests.
 *
 * The core optional features below each have hand-written rules in the guard tests. A registry
 * plugin adds its feature through the `web.features` slot instead, so the guards cannot name it.
 * They apply one convention to every optional feature that is not in this list:
 *
 * - Browser routes delegate to `src/features/<feature>/server/<feature>-route.ts`, which exports
 *   `resolve<Feature>Route()` (availability) and `private<Feature>Response()`, and orders its
 *   internal `resolve<Feature>ReadRoute` and `resolve<Feature>MutationRoute` resolvers
 *   (`mock-bff-route-contract.test.ts`).
 * - Its client message namespace is `<feature>: ['<feature>']`, provided by a console route layout
 *   (`i18n-client-messages.test.tsx`).
 * - Its API error codes use the `<FEATURE>.` namespace (`error-code-vocabulary.test.ts`).
 * - A feature with a browser service ships `src/test/mock-bff-parity/<feature>.ts`
 *   (`mock-bff-service-parity.test.ts`).
 *
 * Adding a feature here exempts it from the convention, so only core features belong here.
 */
export const CORE_OPTIONAL_WEB_FEATURES = [
  'organization',
  'permission',
  'notification',
  'asset',
  'setting',
  'usage',
  'webhook',
] as const;

const core = new Set<string>(CORE_OPTIONAL_WEB_FEATURES);

/** Optional Web features installed by plugins, in selection order. */
export const PLUGIN_WEB_FEATURES: readonly string[] = OPTIONAL_WEB_FEATURES.filter(
  feature => !core.has(feature)
);

/** `help-center` becomes `HelpCenter`, as in `resolveHelpCenterRoute`. */
export function pascalCase(feature: string): string {
  return feature
    .split('-')
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join('');
}

/** `help-center` becomes `HELP_CENTER`, the feature's API error code namespace. */
export function errorNamespace(feature: string): string {
  return feature.toUpperCase().replaceAll('-', '_');
}

/** What `src/test/mock-bff-parity/<feature>.ts` receives from the service parity harness. */
export interface ServiceParityContext {
  /** An organization the mock session user owns. */
  organizationId: number;
}
