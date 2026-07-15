'use client';

import { useState } from 'react';
import Link from 'next/link';
import { AlertCircle, ArrowLeft, Building2, CheckCircle2, RefreshCw } from 'lucide-react';
import { toast } from 'sonner';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { ROUTES } from '@/constants/routes';
import { RoleBadge } from '@/features/organization/components/organization-directory';
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

  return (
    <OrganizationSettings
      key={query.data.organization_id}
      context={query.data}
    />
  );
}

function OrganizationSettings({ context }: { context: OrganizationContext }) {
  const t = useT('organization');
  const common = useT('common');
  const mutation = useUpdateOrganization(context.organization_id);
  const [name, setName] = useState(context.organization_name);
  const [clientError, setClientError] = useState<string>();
  const canRename = context.role === 'owner' || context.role === 'admin';

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const parsed = updateOrganizationSchema.safeParse({ name });
    if (!parsed.success) {
      setClientError(t('nameInvalid'));
      return;
    }
    setClientError(undefined);
    mutation.mutate(parsed.data, {
      onSuccess: (organization) => {
        setName(organization.name);
        toast.success(t('updateSuccess'));
      },
    });
  };

  const nameError = clientError ?? (
    hasOrganizationFieldError(mutation.error, 'name') ? t('nameInvalid') : undefined
  );

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

      <section className="py-6" aria-labelledby="organization-profile-heading">
        <div className="mb-5">
          <h2 id="organization-profile-heading" className="text-base font-semibold">
            {t('profile')}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">{t('profileDescription')}</p>
        </div>

        <form onSubmit={handleSubmit} className="max-w-xl space-y-5" aria-busy={mutation.isPending}>
          {mutation.error ? (
            <Alert variant="destructive">
              <AlertCircle aria-hidden="true" className="size-4" />
              <AlertDescription>{t(resolveOrganizationErrorKey(mutation.error))}</AlertDescription>
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
              disabled={!canRename || mutation.isPending}
              onChange={(event) => {
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
          {canRename ? (
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
