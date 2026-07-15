'use client';

import Link from 'next/link';
import { AlertCircle, ArrowRight, Building2, RefreshCw } from 'lucide-react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { getOrganizationRoute } from '@/constants/routes';
import { AcceptOrganizationInvitationDialog } from '@/features/organization/components/accept-organization-invitation-dialog';
import { CreateOrganizationDialog } from '@/features/organization/components/create-organization-dialog';
import { useOrganizations } from '@/features/organization/hooks/use-organizations';
import type { OrganizationRole } from '@/features/organization/types';
import { resolveOrganizationErrorKey } from '@/features/organization/utils/organization-error';
import { useT } from '@/i18n';

export function OrganizationDirectory() {
  const t = useT('organization');
  const query = useOrganizations();

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
      <div className="flex flex-col gap-4 border-b pb-6 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-1.5">
          <h1 className="text-2xl font-semibold text-foreground">{t('title')}</h1>
          <p className="max-w-2xl text-sm text-muted-foreground">{t('description')}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <AcceptOrganizationInvitationDialog />
          <CreateOrganizationDialog />
        </div>
      </div>

      <section className="pt-6" aria-labelledby="organization-list-heading">
        <h2 id="organization-list-heading" className="sr-only">{t('list')}</h2>
        {query.isPending ? <OrganizationTableSkeleton /> : null}
        {query.error ? (
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
        ) : null}
        {query.data?.items.length === 0 ? (
          <div className="flex min-h-56 flex-col items-center justify-center border border-dashed p-8 text-center">
            <Building2 className="mb-4 size-8 text-muted-foreground" aria-hidden="true" />
            <h2 className="text-base font-medium">{t('emptyTitle')}</h2>
            <p className="mt-1 max-w-md text-sm text-muted-foreground">{t('emptyDescription')}</p>
          </div>
        ) : null}
        {query.data && query.data.items.length > 0 ? (
          <div className="overflow-hidden rounded-md border bg-background">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('name')}</TableHead>
                  <TableHead className="hidden sm:table-cell">{t('slug')}</TableHead>
                  <TableHead>{t('role')}</TableHead>
                  <TableHead className="w-12"><span className="sr-only">{t('open')}</span></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {query.data.items.map((organization) => (
                  <TableRow key={organization.id}>
                    <TableCell className="font-medium text-foreground">
                      <span className="block">{organization.name}</span>
                      <span className="block font-mono text-xs font-normal text-muted-foreground sm:hidden">
                        {organization.slug}
                      </span>
                    </TableCell>
                    <TableCell className="hidden font-mono text-xs text-muted-foreground sm:table-cell">
                      {organization.slug}
                    </TableCell>
                    <TableCell>
                      <RoleBadge role={organization.role} />
                    </TableCell>
                    <TableCell className="text-right">
                      <Button asChild variant="ghost" size="sm" isIcon>
                        <Link
                          href={getOrganizationRoute(organization.id)}
                          aria-label={`${t('open')} ${organization.name}`}
                        >
                          <ArrowRight className="size-4" aria-hidden="true" />
                        </Link>
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : null}
      </section>
    </div>
  );
}

export function RoleBadge({ role }: { role: OrganizationRole }) {
  const t = useT('organization');
  return <Badge variant={role === 'owner' ? 'default' : 'secondary'}>{t(`roles.${role}`)}</Badge>;
}

function OrganizationTableSkeleton() {
  return (
    <div className="space-y-3 rounded-md border p-4" aria-hidden="true">
      <Skeleton className="h-8 w-full" />
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-10 w-full" />
    </div>
  );
}
