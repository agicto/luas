import { readdirSync, readFileSync, statSync } from 'node:fs';
import { relative, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const appApiRoot = resolve(process.cwd(), 'src/app/api');
const routeFileName = 'route.ts';
const routeHandlerPattern =
  /\bexport\s+(?:(?:async\s+)?function\s+(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\s*\(|const\s+(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\s*=)/g;
const unsafeMethods = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);
const organizationRouteModule = resolve(
  process.cwd(),
  'src/features/organization/server/organization-route.ts'
);
const organizationLifecycleRouteModule = resolve(
  process.cwd(),
  'src/features/organization/server/organization-lifecycle-route.ts'
);
const apiKeyRouteModule = resolve(process.cwd(), 'src/features/api-key/server/api-key-route.ts');
const permissionRouteModule = resolve(
  process.cwd(),
  'src/features/permission/server/permission-route.ts'
);
const notificationRouteModule = resolve(
  process.cwd(),
  'src/features/notification/server/notification-route.ts'
);
const assetRouteModule = resolve(process.cwd(), 'src/features/asset/server/asset-route.ts');
const settingRouteModule = resolve(process.cwd(), 'src/features/setting/server/setting-route.ts');

const forbiddenRoutePatterns = [
  {
    label: 'legacy ErrorCode constants',
    pattern: /\bErrorCode\b/,
  },
  {
    label: 'frontend-only ClientErrorCode',
    pattern: /\bClientErrorCode\b/,
  },
  {
    label: 'legacy underscore error code string',
    pattern: /['"`](?:SYS|AUTH|BIZ|VAL|NET|TIMEOUT|UNKNOWN)_\d{3}['"`]/,
  },
] as const;

function listRouteFiles(dir: string): string[] {
  return readdirSync(dir).flatMap(entry => {
    const path = resolve(dir, entry);
    const stat = statSync(path);

    if (stat.isDirectory()) {
      return listRouteFiles(path);
    }

    return entry === routeFileName ? [path] : [];
  });
}

function readRoute(path: string): string {
  return readFileSync(path, 'utf8');
}

function relativeRoute(path: string): string {
  return relative(appApiRoot, path);
}

function productionGuard(path: string): string {
  return relativeRoute(path).startsWith('auth/') ? 'resolveAuthRoute(' : 'guardMockBffRoute(';
}

function delegatesOrganizationGuard(path: string, handler: RouteHandlerSource): boolean {
  const route = relativeRoute(path);
  return (
    (route === 'organization-context/route.ts' ||
      route.startsWith('organization-invitations/') ||
      route.startsWith('organizations/')) &&
    /\b(?:accept|create|get|list|remove|resolve|revoke|transfer|update)Organization[A-Za-z]*Route\b/.test(
      handler.source
    )
  );
}

function delegatesApiKeyGuard(path: string, handler: RouteHandlerSource): boolean {
  return (
    relativeRoute(path).startsWith('api-keys/') &&
    /\b(?:create|list|revoke)ApiKeys?Route\b/.test(handler.source)
  );
}

function isPermissionRoute(path: string): boolean {
  const route = relativeRoute(path);
  return (
    route === 'permission-context/route.ts' ||
    route === 'permissions/route.ts' ||
    route.startsWith('access-roles/') ||
    /^organization-members\/[^/]+\/access-roles\/route\.ts$/.test(route)
  );
}

function delegatesPermissionGuard(path: string, handler: RouteHandlerSource): boolean {
  return (
    isPermissionRoute(path) &&
    /\b(?:create|delete|get|list|replace|update)(?:AccessRole|MemberAccessRole|Permission)[A-Za-z]*Route\b/.test(
      handler.source
    )
  );
}

function isNotificationRoute(path: string): boolean {
  const route = relativeRoute(path);
  return (
    route === 'notifications/route.ts' ||
    route.startsWith('notifications/') ||
    route === 'notification-status/route.ts' ||
    route === 'notification-read-state/route.ts' ||
    route === 'notification-preferences/route.ts'
  );
}

function delegatesNotificationGuard(path: string, handler: RouteHandlerSource): boolean {
  return (
    isNotificationRoute(path) &&
    /\b(?:get|list|mark|replace)Notification[A-Za-z]*Route\b/.test(handler.source)
  );
}

function isAssetRoute(path: string): boolean {
  const route = relativeRoute(path);
  return (
    route === 'assets/route.ts' ||
    route.startsWith('assets/') ||
    route.startsWith('asset-transfers/')
  );
}

function delegatesAssetGuard(path: string, handler: RouteHandlerSource): boolean {
  return (
    isAssetRoute(path) &&
    /\b(?:accept|complete|create|delete|download|list)Asset[A-Za-z]*Route\b/.test(handler.source)
  );
}

function isSettingRoute(path: string): boolean {
  const route = relativeRoute(path);
  return route.startsWith('settings/') || route.startsWith('organization-settings/');
}

function delegatesSettingGuard(path: string, handler: RouteHandlerSource): boolean {
  return (
    isSettingRoute(path) &&
    /\b(?:organizationSettings|publicSettings|resetOrganizationSetting|resetUserSetting|setOrganizationSetting|setUserSetting|userSettings)Route\b/.test(
      handler.source
    )
  );
}

interface RouteHandlerSource {
  method: string;
  source: string;
}

function routeHandlers(path: string): RouteHandlerSource[] {
  const source = readRoute(path);
  const matches = Array.from(source.matchAll(routeHandlerPattern));

  return matches.map((match, index) => ({
    method: match[1] ?? match[2],
    source: source.slice(match.index, matches[index + 1]?.index ?? source.length),
  }));
}

describe('mock BFF route contract', () => {
  const routeFiles = listRouteFiles(appApiRoot).sort((a, b) =>
    relativeRoute(a).localeCompare(relativeRoute(b))
  );

  it('discovers route handlers under src/app/api', () => {
    expect(routeFiles.length).toBeGreaterThan(0);
  });

  it('discovers at least one exported HTTP handler in every route file', () => {
    const offenders = routeFiles
      .filter(path => routeHandlers(path).length === 0)
      .map(relativeRoute);

    expect(offenders).toEqual([]);
  });

  it('keeps every route handler behind its production availability guard', () => {
    const offenders = routeFiles.flatMap(path =>
      routeHandlers(path)
        .filter(
          handler =>
            !handler.source.includes(productionGuard(path)) &&
            !delegatesOrganizationGuard(path, handler) &&
            !delegatesApiKeyGuard(path, handler) &&
            !delegatesPermissionGuard(path, handler) &&
            !delegatesNotificationGuard(path, handler) &&
            !delegatesAssetGuard(path, handler) &&
            !delegatesSettingGuard(path, handler)
        )
        .map(handler => `${relativeRoute(path)}:${handler.method}`)
    );

    expect(offenders).toEqual([]);
  });

  it('keeps every unsafe mock BFF handler behind the same-origin guard', () => {
    const offenders = routeFiles.flatMap(path =>
      routeHandlers(path)
        .filter(handler => unsafeMethods.has(handler.method))
        .filter(handler => !delegatesOrganizationGuard(path, handler))
        .filter(handler => !delegatesApiKeyGuard(path, handler))
        .filter(handler => !delegatesPermissionGuard(path, handler))
        .filter(handler => !delegatesNotificationGuard(path, handler))
        .filter(handler => !delegatesAssetGuard(path, handler))
        .filter(handler => !delegatesSettingGuard(path, handler))
        .filter(handler => {
          const availabilityGuard = handler.source.indexOf(productionGuard(path));
          const originGuard = handler.source.indexOf('guardSameOriginMutation(');

          return originGuard < 0 || availabilityGuard < 0 || originGuard < availabilityGuard;
        })
        .map(handler => `${relativeRoute(path)}:${handler.method}`)
    );

    expect(offenders).toEqual([]);
  });

  it('finalizes every auth route as a private no-store response', () => {
    const offenders = routeFiles
      .filter(path => relativeRoute(path).startsWith('auth/'))
      .flatMap(path =>
        routeHandlers(path)
          .filter(handler => !handler.source.includes('privateAuthResponse('))
          .map(handler => `${relativeRoute(path)}:${handler.method}`)
      );

    expect(offenders).toEqual([]);
  });

  it('finalizes every organization route as a private no-store response', () => {
    const offenders = routeFiles
      .filter(path => {
        const route = relativeRoute(path);
        return (
          route === 'organization-context/route.ts' ||
          route.startsWith('organization-invitations/') ||
          route.startsWith('organizations/')
        );
      })
      .flatMap(path =>
        routeHandlers(path)
          .filter(handler => !handler.source.includes('privateOrganizationResponse('))
          .map(handler => `${relativeRoute(path)}:${handler.method}`)
      );

    expect(offenders).toEqual([]);
  });

  it('finalizes every API key route as a private no-store response', () => {
    const offenders = routeFiles
      .filter(path => relativeRoute(path).startsWith('api-keys/'))
      .flatMap(path =>
        routeHandlers(path)
          .filter(handler => !handler.source.includes('privateApiKeyResponse('))
          .map(handler => `${relativeRoute(path)}:${handler.method}`)
      );

    expect(offenders).toEqual([]);
  });

  it('finalizes every permission route as a private no-store response', () => {
    const offenders = routeFiles.filter(isPermissionRoute).flatMap(path =>
      routeHandlers(path)
        .filter(handler => !handler.source.includes('privateOrganizationResponse('))
        .map(handler => `${relativeRoute(path)}:${handler.method}`)
    );

    expect(offenders).toEqual([]);
  });

  it('finalizes every notification route as a private no-store response', () => {
    const offenders = routeFiles.filter(isNotificationRoute).flatMap(path =>
      routeHandlers(path)
        .filter(handler => !handler.source.includes('privateNotificationResponse('))
        .map(handler => `${relativeRoute(path)}:${handler.method}`)
    );

    expect(offenders).toEqual([]);
  });

  it('finalizes every asset route as a private no-store response', () => {
    const offenders = routeFiles.filter(isAssetRoute).flatMap(path =>
      routeHandlers(path)
        .filter(handler => !handler.source.includes('privateAssetResponse('))
        .map(handler => `${relativeRoute(path)}:${handler.method}`)
    );

    expect(offenders).toEqual([]);
  });

  it('finalizes every private setting route as a private no-store response', () => {
    const offenders = routeFiles
      .filter(path => {
        const route = relativeRoute(path);
        return route.startsWith('settings/user/') || route.startsWith('organization-settings/');
      })
      .flatMap(path =>
        routeHandlers(path)
          .filter(handler => !handler.source.includes('privateSettingResponse('))
          .map(handler => `${relativeRoute(path)}:${handler.method}`)
      );

    expect(offenders).toEqual([]);
  });

  it('keeps delegated setting writes ordered behind availability and origin guards', () => {
    const source = readFileSync(settingRouteModule, 'utf8');
    for (const resolverName of [
      'resolveSettingMutationRoute',
      'resolveOrganizationSettingMutationTarget',
    ]) {
      const resolver = internalFunction(source, resolverName);
      const availabilityGuard = resolver.indexOf('resolveSettingRoute(');
      const originGuard = resolver.indexOf('guardSameOriginMutation(');
      const authentication = resolver.indexOf('resolveAuthenticatedSettingRoute(');

      expect(availabilityGuard, resolverName).toBeGreaterThanOrEqual(0);
      expect(originGuard, resolverName).toBeGreaterThan(availabilityGuard);
      expect(authentication, resolverName).toBeGreaterThan(originGuard);
    }

    const userMutations = ['setUserSettingRoute', 'resetUserSettingRoute'];
    const organizationMutations = ['setOrganizationSettingRoute', 'resetOrganizationSettingRoute'];
    for (const name of userMutations) {
      expect(exportedFunction(source, name), name).toContain('resolveSettingMutationRoute(');
    }
    for (const name of organizationMutations) {
      expect(exportedFunction(source, name), name).toContain(
        'resolveOrganizationSettingMutationTarget('
      );
    }
  });

  it('keeps delegated asset writes ordered behind availability and origin guards', () => {
    const source = readFileSync(assetRouteModule, 'utf8');
    const resolver =
      source
        .split('async function resolveMutationRoute', 2)[1]
        ?.split('async function resolveAuthenticatedRoute', 1)[0] ?? '';
    const availabilityGuard = resolver.indexOf('resolveAssetRoute(');
    const originGuard = resolver.indexOf('guardSameOriginMutation(');
    const authentication = resolver.indexOf('resolveAuthenticatedRoute(');

    expect(availabilityGuard).toBeGreaterThanOrEqual(0);
    expect(originGuard).toBeGreaterThan(availabilityGuard);
    expect(authentication).toBeGreaterThan(originGuard);

    for (const name of [
      'createAssetUploadIntentRoute',
      'completeAssetRoute',
      'createAssetDownloadGrantRoute',
      'deleteAssetRoute',
    ]) {
      const handler = exportedFunction(source, name);
      expect(handler.indexOf('resolveMutationRoute('), name).toBeGreaterThanOrEqual(0);
    }
  });

  it('keeps delegated notification writes ordered behind availability and origin guards', () => {
    const source = readFileSync(notificationRouteModule, 'utf8');
    const resolver =
      source
        .split('async function resolveMutationRoute', 2)[1]
        ?.split('async function resolveAuthenticatedRoute', 1)[0] ?? '';
    const availabilityGuard = resolver.indexOf('resolveNotificationRoute(');
    const originGuard = resolver.indexOf('guardSameOriginMutation(');
    const authentication = resolver.indexOf('resolveAuthenticatedRoute(');

    expect(availabilityGuard).toBeGreaterThanOrEqual(0);
    expect(originGuard).toBeGreaterThan(availabilityGuard);
    expect(authentication).toBeGreaterThan(originGuard);

    for (const name of [
      'replaceNotificationReadStateRoute',
      'markNotificationsReadRoute',
      'replaceNotificationPreferenceRoute',
    ]) {
      const handler = exportedFunction(source, name);
      const mutationGuard = handler.indexOf('resolveMutationRoute(');
      const bodyRead = handler.indexOf('readJsonBody(');

      expect(mutationGuard, name).toBeGreaterThanOrEqual(0);
      expect(bodyRead, name).toBeGreaterThan(mutationGuard);
    }
  });

  it('keeps delegated permission writes ordered behind availability and origin guards', () => {
    const source = readFileSync(permissionRouteModule, 'utf8');
    const resolver =
      source
        .split('async function resolveMutationRoute', 2)[1]
        ?.split('async function resolveAuthenticatedRoute', 1)[0] ?? '';
    const availabilityGuard = resolver.indexOf('resolvePermissionRoute(');
    const originGuard = resolver.indexOf('guardSameOriginMutation(');
    const authentication = resolver.indexOf('resolveAuthenticatedRoute(');

    expect(availabilityGuard).toBeGreaterThanOrEqual(0);
    expect(originGuard).toBeGreaterThan(availabilityGuard);
    expect(authentication).toBeGreaterThan(originGuard);

    for (const name of [
      'createAccessRoleRoute',
      'updateAccessRoleRoute',
      'deleteAccessRoleRoute',
      'replaceMemberAccessRolesRoute',
    ]) {
      const handler = exportedFunction(source, name);
      const mutationGuard = handler.indexOf('resolveMutationRoute(');
      const bodyRead = handler.indexOf('readJsonBody(');

      expect(mutationGuard, name).toBeGreaterThanOrEqual(0);
      if (bodyRead >= 0) {
        expect(bodyRead, name).toBeGreaterThan(mutationGuard);
      }
    }
  });

  it('keeps delegated organization writes ordered behind availability and origin guards', () => {
    const source = readFileSync(organizationRouteModule, 'utf8');
    const nextFunction = 'export async function getOrganizationRoute';
    const create =
      source
        .split('export async function createOrganizationRoute', 2)[1]
        ?.split(nextFunction, 1)[0] ?? '';
    const update =
      source
        .split('export async function updateOrganizationRoute', 2)[1]
        ?.split('export async function resolveOrganizationContextRoute', 1)[0] ?? '';

    for (const handler of [create, update]) {
      const availabilityGuard = handler.indexOf('resolveOrganizationRoute(');
      const originGuard = handler.indexOf('guardSameOriginMutation(');
      const bodyRead = handler.indexOf('readJsonBody(');

      expect(availabilityGuard).toBeGreaterThanOrEqual(0);
      expect(originGuard).toBeGreaterThan(availabilityGuard);
      expect(bodyRead).toBeGreaterThan(originGuard);
    }

    const lifecycleSource = readFileSync(organizationLifecycleRouteModule, 'utf8');
    for (const name of [
      'updateOrganizationMemberRoute',
      'removeOrganizationMemberRoute',
      'transferOrganizationOwnershipRoute',
      'createOrganizationInvitationRoute',
      'revokeOrganizationInvitationRoute',
      'acceptOrganizationInvitationRoute',
    ]) {
      const handler = exportedFunction(lifecycleSource, name);
      const availabilityGuard = handler.indexOf('resolveOrganizationRoute(');
      const originGuard = handler.indexOf('guardSameOriginMutation(');
      const authentication = handler.indexOf('authenticateOrganizationBackend(');
      const bodyRead = handler.indexOf('readJsonBody(');

      expect(availabilityGuard, name).toBeGreaterThanOrEqual(0);
      expect(originGuard, name).toBeGreaterThan(availabilityGuard);
      expect(authentication, name).toBeGreaterThan(originGuard);
      if (bodyRead >= 0) {
        expect(bodyRead, name).toBeGreaterThan(authentication);
      }
    }
  });

  it('keeps delegated API key writes ordered behind availability and origin guards', () => {
    const source = readFileSync(apiKeyRouteModule, 'utf8');
    for (const name of ['createApiKeyRoute', 'revokeApiKeyRoute']) {
      const handler = exportedFunction(source, name);
      const availabilityGuard = handler.indexOf('resolveApiKeyRoute(');
      const originGuard = handler.indexOf('guardSameOriginMutation(');
      const authentication = handler.indexOf('authenticateApiKeyBackend(');
      const bodyRead = handler.indexOf('readJsonBody(');

      expect(availabilityGuard, name).toBeGreaterThanOrEqual(0);
      expect(originGuard, name).toBeGreaterThan(availabilityGuard);
      expect(authentication, name).toBeGreaterThan(originGuard);
      if (bodyRead >= 0) expect(bodyRead, name).toBeGreaterThan(authentication);
    }
  });

  it('keeps mock route JSON envelopes behind shared response helpers', () => {
    const offenders = routeFiles
      .filter(path => /\bNextResponse\.json\s*\(/.test(readRoute(path)))
      .map(relativeRoute);

    expect(offenders).toEqual([]);
  });

  it('keeps mock route errors on canonical API error codes', () => {
    const offenders = routeFiles.flatMap(path => {
      const source = readRoute(path);

      return forbiddenRoutePatterns
        .filter(({ pattern }) => pattern.test(source))
        .map(({ label }) => `${relativeRoute(path)}: ${label}`);
    });

    expect(offenders).toEqual([]);
  });
});

function exportedFunction(source: string, name: string): string {
  const marker = `export async function ${name}`;
  const start = source.indexOf(marker);
  if (start < 0) return '';
  const next = source.indexOf('export async function ', start + marker.length);
  return source.slice(start, next < 0 ? source.length : next);
}

function internalFunction(source: string, name: string): string {
  const marker = `async function ${name}`;
  const start = source.indexOf(marker);
  if (start < 0) return '';
  const next = source.indexOf('async function ', start + marker.length);
  return source.slice(start, next < 0 ? source.length : next);
}
