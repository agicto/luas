'use client';

import { useState } from 'react';
import { useLocale } from 'next-intl';
import { AlertCircle, RefreshCw, Trash2 } from 'lucide-react';
import { toast } from 'sonner';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogBody,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { CreateOrganizationInvitationDialog } from '@/features/organization/components/create-organization-invitation-dialog';
import {
  useOrganizationInvitations,
  useRevokeOrganizationInvitation,
} from '@/features/organization/hooks/use-organizations';
import type {
  OrganizationInvitation,
  OrganizationInvitationStatus,
} from '@/features/organization/types';
import { formatOrganizationDate } from '@/features/organization/utils/format-organization-date';
import { resolveOrganizationErrorKey } from '@/features/organization/utils/organization-error';
import { useT } from '@/i18n';

export function OrganizationInvitations({
  organizationId,
}: {
  organizationId: number;
}) {
  const t = useT('organization');
  const common = useT('common');
  const locale = useLocale();
  const query = useOrganizationInvitations(organizationId);
  const mutation = useRevokeOrganizationInvitation(organizationId);
  const [selected, setSelected] = useState<OrganizationInvitation>();

  const revoke = () => {
    if (!selected) return;
    mutation.mutate(selected.id, {
      onSuccess: () => {
        toast.success(t('revokeInvitationSuccess'));
        setSelected(undefined);
      },
    });
  };

  return (
    <section className="py-6" aria-labelledby="organization-invitations-heading">
      <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 id="organization-invitations-heading" className="text-base font-semibold">
            {t('invitationsTitle')}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {t('invitationsDescription')}
          </p>
        </div>
        <CreateOrganizationInvitationDialog organizationId={organizationId} />
      </div>

      {query.isPending ? <InvitationTableSkeleton /> : null}
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
        <div className="border border-dashed px-4 py-10 text-center text-sm text-muted-foreground">
          {t('noInvitations')}
        </div>
      ) : null}
      {query.data && query.data.items.length > 0 ? (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('email')}</TableHead>
                <TableHead>{t('role')}</TableHead>
                <TableHead>{t('status')}</TableHead>
                <TableHead className="hidden lg:table-cell">{t('expiresAt')}</TableHead>
                <TableHead className="w-32 text-right">
                  <span className="sr-only">{t('actions')}</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {query.data.items.map((invitation) => (
                <TableRow key={invitation.id}>
                  <TableCell className="font-medium">{invitation.email}</TableCell>
                  <TableCell>{t(`roles.${invitation.role}`)}</TableCell>
                  <TableCell>
                    <InvitationStatusBadge status={invitation.status} />
                  </TableCell>
                  <TableCell className="hidden text-sm text-muted-foreground lg:table-cell">
                    {formatOrganizationDate(invitation.expires_at, locale)}
                  </TableCell>
                  <TableCell className="text-right">
                    {invitation.status === 'pending' ? (
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        icon={<Trash2 className="size-3.5" />}
                        onClick={() => {
                          mutation.reset();
                          setSelected(invitation);
                        }}
                      >
                        {t('revokeInvitation')}
                      </Button>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}

      <Dialog
        open={Boolean(selected)}
        onOpenChange={(open) => {
          if (!open && !mutation.isPending) {
            setSelected(undefined);
            mutation.reset();
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('revokeInvitationTitle')}</DialogTitle>
            <DialogDescription>{t('revokeInvitationDescription')}</DialogDescription>
          </DialogHeader>
          <DialogBody>
            {mutation.error ? (
              <Alert variant="destructive">
                <AlertDescription>
                  {t(resolveOrganizationErrorKey(mutation.error))}
                </AlertDescription>
              </Alert>
            ) : null}
            <p className="text-sm font-medium text-foreground">{selected?.email}</p>
          </DialogBody>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="outline" disabled={mutation.isPending}>
                {common('cancel')}
              </Button>
            </DialogClose>
            <Button
              type="button"
              variant="destructive"
              loading={mutation.isPending}
              onClick={revoke}
            >
              {t('revokeInvitation')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  );
}

function InvitationStatusBadge({
  status,
}: {
  status: OrganizationInvitationStatus;
}) {
  const t = useT('organization');
  const variant = status === 'pending' ? 'default' : status === 'expired' ? 'outline' : 'secondary';
  return <Badge variant={variant}>{t(`invitationStatus.${status}`)}</Badge>;
}

function InvitationTableSkeleton() {
  return (
    <div className="space-y-3 rounded-md border p-4" aria-hidden="true">
      <Skeleton className="h-8 w-full" />
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-10 w-full" />
    </div>
  );
}
