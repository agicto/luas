'use client';

import { lazy, Suspense, useState } from 'react';
import Link from 'next/link';
import {
  AlertCircle,
  ArrowLeft,
  Building2,
  CheckCircle2,
  Mail,
  Gauge,
  RefreshCw,
  Settings2,
  ShieldCheck,
  SlidersHorizontal,
  Users,
} from 'lucide-react';
import { toast } from 'sonner';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ROUTES } from '@/constants/routes';
import { RoleBadge } from '@/features/organization/components/organization-directory';
import { OrganizationInvitations } from '@/features/organization/components/organization-invitations';
import { OrganizationMembers } from '@/features/organization/components/organization-members';
import {
  useOrganizationContext,
  useUpdateOrganization,
} from '@/features/organization/hooks/use-organizations';
import { updateOrganizationSchema } from '@/features/organization/schemas';
import type { OrganizationContext } from '@/features/organization/types';
import {
  hasOrganizationFieldError,
  resolveOrganizationErrorKey,
} from '@/features/organization/utils/organization-error';
import { useT } from '@/i18n';
import { isWebFeatureEnabled } from '@/config/features';

const PermissionManagement = lazy(async () => {
  const feature = await import('@/features/permission/components/permission-management');
  return { default: feature.PermissionManagement };
});

const OrganizationSettingPanel = lazy(async () => {
  const feature = await import('@/features/setting/components/organization-setting-panel');
  return { default: feature.OrganizationSettingPanel };
});

const OrganizationUsagePanel = lazy(async () => {
  const feature = await import('@/features/usage/components/usage-panel');
  return { default: feature.OrganizationUsagePanel };
});

export function OrganizationOverview({ organizationId }: { organizationId: number }) {
  const t = useT('organization');
  const query = useOrganizationContext(organizationId);

  if (query.isPending) {
    return <OrganizationOverviewSkeleton />;
  }
  if (query.error || !query.data) {
    return (
      <div className="mx-auto w-full max-w-4xl px-4 py-8 sm:px-6">
        <Alert variant="destructive">
          <AlertCircle aria-hidden="true" className="size-4" />
          <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
            <span>{t(resolveOrganizationErrorKey(query.error))}</span>
            <Button
              type="button"
              size="sm"
              variant="outline"
              icon={<RefreshCw className="size-3.5" />}
              onClick={() => query.refetch()}
            >
              {t('retry')}
            </Button>
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  return <OrganizationSettings key={query.data.organization_id} context={query.data} />;
}

function OrganizationSettings({ context }: { context: OrganizationContext }) {
  const t = useT('organization');
  const common = useT('common');
  const mutation = useUpdateOrganization(context.organization_id);
  const [name, setName] = useState(context.organization_name);
  const [clientError, setClientError] = useState<string>();
  const canManageOrganization = context.role === 'owner' || context.role === 'admin';
  const permissionEnabled = isWebFeatureEnabled('permission');
  const settingEnabled = isWebFeatureEnabled('setting');
  const usageEnabled = isWebFeatureEnabled('usage');

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const parsed = updateOrganizationSchema.safeParse({ name });
    if (!parsed.success) {
      setClientError(t('nameInvalid'));
      return;
    }
    setClientError(undefined);
    mutation.mutate(parsed.data, {
      onSuccess: organization => {
        setName(organization.name);
        toast.success(t('updateSuccess'));
      },
    });
  };

  const nameError =
    clientError ??
    (hasOrganizationFieldError(mutation.error, 'name') ? t('nameInvalid') : undefined);

  return (
    <div className="mx-auto w-full max-w-4xl px-4 py-6 sm:px-6 lg:py-8">
      <Button asChild variant="link" size="sm" className="mb-4 px-0">
        <Link href={ROUTES.CONSOLE.ORGANIZATIONS}>
          <ArrowLeft className="size-4" aria-hidden="true" />
          {t('back')}
        </Link>
      </Button>

      <div className="flex flex-col gap-5 border-b pb-6 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 items-start gap-3">
          <div className="flex size-10 shrink-0 items-center justify-center rounded-md border bg-muted">
            <Building2 className="size-5 text-muted-foreground" aria-hidden="true" />
          </div>
          <div className="min-w-0">
            <h1 className="truncate text-2xl font-semibold text-foreground">
              {context.organization_name}
            </h1>
            <p className="mt-1 font-mono text-xs text-muted-foreground">
              {context.organization_slug}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <RoleBadge role={context.role} />
          <span className="inline-flex items-center gap-1.5 text-xs font-medium text-success">
            <CheckCircle2 className="size-3.5" aria-hidden="true" />
            {t('contextVerified')}
          </span>
        </div>
      </div>

      <Tabs defaultValue="profile" className="pt-5">
        <TabsList className="max-w-full overflow-x-auto">
          <TabsTrigger value="profile">
            <Settings2 aria-hidden="true" />
            {t('tabs.profile')}
          </TabsTrigger>
          <TabsTrigger value="members">
            <Users aria-hidden="true" />
            {t('tabs.members')}
          </TabsTrigger>
          {canManageOrganization ? (
            <TabsTrigger value="invitations">
              <Mail aria-hidden="true" />
              {t('tabs.invitations')}
            </TabsTrigger>
          ) : null}
          {permissionEnabled ? (
            <TabsTrigger value="permissions">
              <ShieldCheck aria-hidden="true" />
              {t('tabs.permissions')}
            </TabsTrigger>
          ) : null}
          {settingEnabled ? (
            <TabsTrigger value="settings">
              <SlidersHorizontal aria-hidden="true" />
              {t('tabs.settings')}
            </TabsTrigger>
          ) : null}
          {usageEnabled && canManageOrganization ? (
            <TabsTrigger value="usage">
              <Gauge aria-hidden="true" />
              {t('tabs.usage')}
            </TabsTrigger>
          ) : null}
        </TabsList>

        <TabsContent value="profile">
          <section className="py-6" aria-labelledby="organization-profile-heading">
            <div className="mb-5">
              <h2 id="organization-profile-heading" className="text-base font-semibold">
                {t('profile')}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">{t('profileDescription')}</p>
            </div>

            <form
              onSubmit={handleSubmit}
              className="max-w-xl space-y-5"
              aria-busy={mutation.isPending}
            >
              {mutation.error ? (
                <Alert variant="destructive">
                  <AlertCircle aria-hidden="true" className="size-4" />
                  <AlertDescription>
                    {t(resolveOrganizationErrorKey(mutation.error))}
                  </AlertDescription>
                </Alert>
              ) : null}
              <div className="space-y-2">
                <Label htmlFor="organization-profile-name">{t('name')}</Label>
                <Input
                  id="organization-profile-name"
                  name="name"
                  autoComplete="organization"
                  required
                  minLength={2}
                  maxLength={200}
                  value={name}
                  errorText={nameError}
                  disabled={!canManageOrganization || mutation.isPending}
                  onChange={event => {
                    setName(event.target.value);
                    setClientError(undefined);
                    mutation.reset();
                  }}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="organization-profile-slug">{t('slug')}</Label>
                <Input
                  id="organization-profile-slug"
                  value={context.organization_slug}
                  disabled
                  readOnly
                />
              </div>
              {canManageOrganization ? (
                <Button
                  type="submit"
                  loading={mutation.isPending}
                  disabled={name.trim() === context.organization_name}
                >
                  {common('save')}
                </Button>
              ) : null}
            </form>
          </section>
        </TabsContent>

        <TabsContent value="members">
          <OrganizationMembers context={context} />
        </TabsContent>

        {canManageOrganization ? (
          <TabsContent value="invitations">
            <OrganizationInvitations organizationId={context.organization_id} />
          </TabsContent>
        ) : null}

        {permissionEnabled ? (
          <TabsContent value="permissions">
            <Suspense fallback={<OrganizationOverviewSkeleton />}>
              <PermissionManagement context={context} />
            </Suspense>
          </TabsContent>
        ) : null}

        {settingEnabled ? (
          <TabsContent value="settings">
            <Suspense fallback={<OrganizationOverviewSkeleton />}>
              <OrganizationSettingPanel
                organizationId={context.organization_id}
                canManage={canManageOrganization}
              />
            </Suspense>
          </TabsContent>
        ) : null}

        {usageEnabled && canManageOrganization ? (
          <TabsContent value="usage">
            <Suspense fallback={<OrganizationOverviewSkeleton />}>
              <OrganizationUsagePanel organizationId={context.organization_id} />
            </Suspense>
          </TabsContent>
        ) : null}
      </Tabs>
    </div>
  );
}

function OrganizationOverviewSkeleton() {
  return (
    <div className="mx-auto w-full max-w-4xl space-y-5 px-4 py-8 sm:px-6" aria-hidden="true">
      <Skeleton className="h-8 w-32" />
      <Skeleton className="h-20 w-full" />
      <Skeleton className="h-10 w-full max-w-xl" />
      <Skeleton className="h-10 w-full max-w-xl" />
    </div>
  );
}
