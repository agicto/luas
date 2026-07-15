'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useLocale } from 'next-intl';
import {
  AlertCircle,
  ArrowRightLeft,
  LogOut,
  MoreHorizontal,
  RefreshCw,
  UserMinus,
} from 'lucide-react';
import { toast } from 'sonner';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ROUTES } from '@/constants/routes';
import { RoleBadge } from '@/features/organization/components/organization-directory';
import {
  useOrganizationMembers,
  useRemoveOrganizationMember,
  useTransferOrganizationOwnership,
  useUpdateOrganizationMember,
} from '@/features/organization/hooks/use-organizations';
import type {
  OrganizationContext,
  OrganizationInvitationRole,
  OrganizationMember,
} from '@/features/organization/types';
import { formatOrganizationDate } from '@/features/organization/utils/format-organization-date';
import { resolveOrganizationErrorKey } from '@/features/organization/utils/organization-error';
import { useT } from '@/i18n';

type PendingMemberAction =
  | { kind: 'leave'; member: OrganizationMember }
  | { kind: 'remove'; member: OrganizationMember }
  | { kind: 'transfer'; member: OrganizationMember };

export function OrganizationMembers({ context }: { context: OrganizationContext }) {
  const t = useT('organization');
  const common = useT('common');
  const locale = useLocale();
  const router = useRouter();
  const query = useOrganizationMembers(context.organization_id);
  const roleMutation = useUpdateOrganizationMember(context.organization_id);
  const removeMutation = useRemoveOrganizationMember(context.organization_id);
  const transferMutation = useTransferOrganizationOwnership(context.organization_id);
  const [pendingAction, setPendingAction] = useState<PendingMemberAction>();

  const closeAction = () => {
    setPendingAction(undefined);
    removeMutation.reset();
    transferMutation.reset();
  };

  const confirmAction = () => {
    if (!pendingAction) return;
    if (pendingAction.kind === 'transfer') {
      transferMutation.mutate(
        { new_owner_member_id: pendingAction.member.id },
        {
          onSuccess: () => {
            toast.success(t('transferOwnershipSuccess'));
            closeAction();
          },
        }
      );
      return;
    }

    removeMutation.mutate(pendingAction.member.id, {
      onSuccess: () => {
        const leaving = pendingAction.kind === 'leave';
        toast.success(
          t(leaving ? 'leaveOrganizationSuccess' : 'removeMemberSuccess')
        );
        closeAction();
        if (leaving) router.push(ROUTES.CONSOLE.ORGANIZATIONS);
      },
    });
  };

  const mutationError =
    roleMutation.error ?? removeMutation.error ?? transferMutation.error;

  return (
    <section className="py-6" aria-labelledby="organization-members-heading">
      <div className="mb-5">
        <h2 id="organization-members-heading" className="text-base font-semibold">
          {t('membersTitle')}
        </h2>
        <p className="mt-1 text-sm text-muted-foreground">
          {t('membersDescription')}
        </p>
      </div>

      {mutationError && !pendingAction ? (
        <Alert variant="destructive" className="mb-4">
          <AlertCircle aria-hidden="true" className="size-4" />
          <AlertDescription>
            {t(resolveOrganizationErrorKey(mutationError))}
          </AlertDescription>
        </Alert>
      ) : null}
      {query.isPending ? <MemberTableSkeleton /> : null}
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
          {t('noMembers')}
        </div>
      ) : null}
      {query.data && query.data.items.length > 0 ? (
        <div className="overflow-x-auto rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('member')}</TableHead>
                <TableHead>{t('role')}</TableHead>
                <TableHead className="hidden md:table-cell">{t('joinedAt')}</TableHead>
                <TableHead className="w-14 text-right">
                  <span className="sr-only">{t('actions')}</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {query.data.items.map((member) => (
                <TableRow key={member.id}>
                  <TableCell>
                    <div className="flex min-w-52 items-center gap-3">
                      <Avatar>
                        {member.avatar ? (
                          <AvatarImage
                            src={member.avatar}
                            alt={member.nickname || member.username}
                          />
                        ) : null}
                        <AvatarFallback className="text-xs font-medium">
                          {memberInitials(member)}
                        </AvatarFallback>
                      </Avatar>
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="truncate text-sm font-medium">
                            {member.nickname || member.username}
                          </span>
                          {member.id === context.membership_id ? (
                            <span className="text-xs text-muted-foreground">{t('you')}</span>
                          ) : null}
                        </div>
                        <p className="truncate text-xs text-muted-foreground">
                          @{member.username}
                        </p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    {canChangeRole(context, member) ? (
                      <Select
                        value={member.role}
                        disabled={
                          roleMutation.isPending &&
                          roleMutation.variables?.memberId === member.id
                        }
                        onValueChange={(role) => {
                          roleMutation.reset();
                          roleMutation.mutate({
                            memberId: member.id,
                            input: { role: role as OrganizationInvitationRole },
                          }, {
                            onSuccess: () => toast.success(t('roleUpdateSuccess')),
                          });
                        }}
                      >
                        <SelectTrigger
                          size="sm"
                          aria-label={`${t('changeRole')}: ${member.username}`}
                        >
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="member">{t('roles.member')}</SelectItem>
                          <SelectItem value="admin">{t('roles.admin')}</SelectItem>
                        </SelectContent>
                      </Select>
                    ) : (
                      <RoleBadge role={member.role} />
                    )}
                  </TableCell>
                  <TableCell className="hidden text-sm text-muted-foreground md:table-cell">
                    {formatOrganizationDate(member.joined_at, locale)}
                  </TableCell>
                  <TableCell className="text-right">
                    <MemberActions
                      context={context}
                      member={member}
                      onAction={setPendingAction}
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}

      <MemberActionDialog
        action={pendingAction}
        error={pendingAction?.kind === 'transfer' ? transferMutation.error : removeMutation.error}
        loading={transferMutation.isPending || removeMutation.isPending}
        onClose={closeAction}
        onConfirm={confirmAction}
        commonCancel={common('cancel')}
      />
    </section>
  );
}

function MemberActions({
  context,
  member,
  onAction,
}: {
  context: OrganizationContext;
  member: OrganizationMember;
  onAction: (action: PendingMemberAction) => void;
}) {
  const t = useT('organization');
  const self = member.id === context.membership_id;
  const canLeave = self && member.role !== 'owner';
  const canRemove =
    !self &&
    ((context.role === 'owner' && member.role !== 'owner') ||
      (context.role === 'admin' && member.role === 'member'));
  const canTransfer =
    context.role === 'owner' && !self && member.role !== 'owner';

  if (!canLeave && !canRemove && !canTransfer) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          isIcon
          noScale
          title={t('actions')}
          aria-label={`${t('actions')}: ${member.username}`}
        >
          <MoreHorizontal className="size-4" aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {canTransfer ? (
          <DropdownMenuItem onSelect={() => onAction({ kind: 'transfer', member })}>
            <ArrowRightLeft />
            {t('transferOwnership')}
          </DropdownMenuItem>
        ) : null}
        {canRemove ? (
          <DropdownMenuItem
            variant="destructive"
            onSelect={() => onAction({ kind: 'remove', member })}
          >
            <UserMinus />
            {t('removeMember')}
          </DropdownMenuItem>
        ) : null}
        {canLeave ? (
          <DropdownMenuItem
            variant="destructive"
            onSelect={() => onAction({ kind: 'leave', member })}
          >
            <LogOut />
            {t('leaveOrganization')}
          </DropdownMenuItem>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function MemberActionDialog({
  action,
  commonCancel,
  error,
  loading,
  onClose,
  onConfirm,
}: {
  action?: PendingMemberAction;
  commonCancel: string;
  error: unknown;
  loading: boolean;
  onClose: () => void;
  onConfirm: () => void;
}) {
  const t = useT('organization');
  const title = action?.kind === 'transfer'
    ? 'transferOwnershipTitle'
    : action?.kind === 'leave'
      ? 'leaveOrganizationTitle'
      : 'removeMemberTitle';
  const description = action?.kind === 'transfer'
    ? 'transferOwnershipDescription'
    : action?.kind === 'leave'
      ? 'leaveOrganizationDescription'
      : 'removeMemberDescription';
  const command = action?.kind === 'transfer'
    ? 'transferOwnership'
    : action?.kind === 'leave'
      ? 'leaveOrganization'
      : 'removeMember';

  return (
    <Dialog
      open={Boolean(action)}
      onOpenChange={(open) => {
        if (!open && !loading) onClose();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t(title)}</DialogTitle>
          <DialogDescription>{t(description)}</DialogDescription>
        </DialogHeader>
        <DialogBody className="space-y-3">
          {error ? (
            <Alert variant="destructive">
              <AlertDescription>{t(resolveOrganizationErrorKey(error))}</AlertDescription>
            </Alert>
          ) : null}
          <p className="text-sm font-medium">
            {action?.member.nickname || action?.member.username}
          </p>
        </DialogBody>
        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="outline" disabled={loading}>
              {commonCancel}
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant={action?.kind === 'transfer' ? 'default' : 'destructive'}
            loading={loading}
            onClick={onConfirm}
          >
            {t(command)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function canChangeRole(
  context: OrganizationContext,
  member: OrganizationMember
): boolean {
  return (
    context.role === 'owner' &&
    member.id !== context.membership_id &&
    member.role !== 'owner'
  );
}

function memberInitials(member: OrganizationMember): string {
  const name = member.nickname || member.username;
  return Array.from(name).slice(0, 2).join('').toUpperCase();
}

function MemberTableSkeleton() {
  return (
    <div className="space-y-3 rounded-md border p-4" aria-hidden="true">
      <Skeleton className="h-8 w-full" />
      <Skeleton className="h-12 w-full" />
      <Skeleton className="h-12 w-full" />
      <Skeleton className="h-12 w-full" />
    </div>
  );
}
