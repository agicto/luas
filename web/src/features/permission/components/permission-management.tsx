'use client';

import { useEffect, useState } from 'react';
import { Pencil, Plus, RefreshCw, ShieldCheck, Trash2, UserCog } from 'lucide-react';
import { toast } from 'sonner';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useOrganizationMembers } from '@/features/organization/hooks/use-organizations';
import type { OrganizationContext, OrganizationMember } from '@/features/organization/types';
import { PermissionKey } from '@/features/permission/constants';
import {
  useAccessRoles,
  useCreateAccessRole,
  useDeleteAccessRole,
  useMemberAccessRoles,
  usePermissionCatalog,
  usePermissionContext,
  useReplaceMemberAccessRoles,
  useUpdateAccessRole,
} from '@/features/permission/hooks/use-permissions';
import {
  createAccessRoleSchema,
  updateAccessRoleSchema,
} from '@/features/permission/schemas';
import type { AccessRole, PermissionContext } from '@/features/permission/types';
import { useT } from '@/i18n';

export function PermissionManagement({ context }: { context: OrganizationContext }) {
  const t = useT('permission');
  const effectiveQuery = usePermissionContext(context.organization_id);
  const effective = effectiveQuery.data;
  const canReadRoles = hasPermission(effective, PermissionKey.ROLES_READ);
  const canManageRoles = hasPermission(effective, PermissionKey.ROLES_MANAGE);
  const canReadAssignments = hasPermission(effective, PermissionKey.ASSIGNMENTS_READ);
  const canManageAssignments = hasPermission(
    effective,
    PermissionKey.ASSIGNMENTS_MANAGE
  );
  const catalogQuery = usePermissionCatalog(context.organization_id, canReadRoles);
  const rolesQuery = useAccessRoles(context.organization_id, canReadRoles);
  const membersQuery = useOrganizationMembers(
    context.organization_id,
    canReadAssignments
  );
  const [editingRole, setEditingRole] = useState<AccessRole | null | undefined>();
  const [deletingRole, setDeletingRole] = useState<AccessRole>();
  const [assigningMember, setAssigningMember] = useState<OrganizationMember>();

  if (effectiveQuery.isPending) {
    return <PermissionManagementSkeleton />;
  }
  if (effectiveQuery.error || !effective) {
    return (
      <PermissionAlert
        message={t('loadError')}
        onRetry={() => effectiveQuery.refetch()}
      />
    );
  }

  return (
    <section className="space-y-8 py-6" aria-labelledby="permission-heading">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 id="permission-heading" className="text-base font-semibold">
            {t('title')}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {t('description')}
          </p>
        </div>
        {canManageRoles ? (
          <Button
            type="button"
            size="sm"
            icon={<Plus className="size-4" aria-hidden="true" />}
            onClick={() => setEditingRole(null)}
          >
            {t('createRole')}
          </Button>
        ) : null}
      </div>

      {!canReadRoles ? (
        <Alert>
          <ShieldCheck aria-hidden="true" className="size-4" />
          <AlertDescription>{t('readDenied')}</AlertDescription>
        </Alert>
      ) : rolesQuery.error || catalogQuery.error ? (
        <PermissionAlert
          message={t('loadError')}
          onRetry={() => {
            rolesQuery.refetch();
            catalogQuery.refetch();
          }}
        />
      ) : rolesQuery.isPending || catalogQuery.isPending ? (
        <PermissionManagementSkeleton />
      ) : (
        <AccessRoleTable
          roles={rolesQuery.data?.items ?? []}
          canManage={canManageRoles}
          onEdit={(role) => setEditingRole(role)}
          onDelete={(role) => setDeletingRole(role)}
        />
      )}

      {canReadAssignments ? (
        <div className="space-y-4 border-t pt-6">
          <div>
            <h3 className="text-sm font-semibold">{t('memberAssignments')}</h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {t('memberAssignmentsDescription')}
            </p>
          </div>
          {membersQuery.error ? (
            <PermissionAlert
              message={t('membersLoadError')}
              onRetry={() => membersQuery.refetch()}
            />
          ) : membersQuery.isPending ? (
            <Skeleton className="h-28 w-full" />
          ) : (
            <MemberAssignmentTable
              members={membersQuery.data?.items ?? []}
              canManage={canManageAssignments}
              onManage={(member) => setAssigningMember(member)}
            />
          )}
        </div>
      ) : null}

      <AccessRoleDialog
        organizationId={context.organization_id}
        role={editingRole}
        catalog={catalogQuery.data?.permissions ?? []}
        open={editingRole !== undefined}
        onOpenChange={(open) => {
          if (!open) setEditingRole(undefined);
        }}
      />
      <DeleteAccessRoleDialog
        organizationId={context.organization_id}
        role={deletingRole}
        onOpenChange={(open) => {
          if (!open) setDeletingRole(undefined);
        }}
      />
      <MemberAccessRoleDialog
        organizationId={context.organization_id}
        member={assigningMember}
        roles={rolesQuery.data?.items ?? []}
        canManage={canManageAssignments}
        onOpenChange={(open) => {
          if (!open) setAssigningMember(undefined);
        }}
      />
    </section>
  );
}

function AccessRoleTable({
  roles,
  canManage,
  onEdit,
  onDelete,
}: {
  roles: AccessRole[];
  canManage: boolean;
  onEdit: (role: AccessRole) => void;
  onDelete: (role: AccessRole) => void;
}) {
  const t = useT('permission');
  if (roles.length === 0) {
    return <p className="py-6 text-sm text-muted-foreground">{t('emptyRoles')}</p>;
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('role')}</TableHead>
          <TableHead>{t('permissions')}</TableHead>
          {canManage ? <TableHead className="w-24 text-right">{t('actions')}</TableHead> : null}
        </TableRow>
      </TableHeader>
      <TableBody>
        {roles.map((role) => (
          <TableRow key={role.id}>
            <TableCell>
              <div className="min-w-44">
                <div className="font-medium">{role.name}</div>
                <div className="mt-0.5 font-mono text-xs text-muted-foreground">
                  {role.slug}
                </div>
              </div>
            </TableCell>
            <TableCell className="whitespace-normal">
              <div className="flex max-w-xl flex-wrap gap-1.5">
                {role.permissions.length === 0 ? (
                  <span className="text-xs text-muted-foreground">{t('noPermissions')}</span>
                ) : role.permissions.map((permission) => (
                  <Badge key={permission} variant="outline" className="font-mono font-normal">
                    {permission}
                  </Badge>
                ))}
              </div>
            </TableCell>
            {canManage ? (
              <TableCell className="text-right">
                <div className="flex justify-end gap-1">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    isIcon
                    title={t('editRole')}
                    onClick={() => onEdit(role)}
                  >
                    <Pencil className="size-4" aria-hidden="true" />
                    <span className="sr-only">{t('editRole')}</span>
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    isIcon
                    title={t('deleteRole')}
                    onClick={() => onDelete(role)}
                  >
                    <Trash2 className="size-4 text-error" aria-hidden="true" />
                    <span className="sr-only">{t('deleteRole')}</span>
                  </Button>
                </div>
              </TableCell>
            ) : null}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function MemberAssignmentTable({
  members,
  canManage,
  onManage,
}: {
  members: OrganizationMember[];
  canManage: boolean;
  onManage: (member: OrganizationMember) => void;
}) {
  const t = useT('permission');
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('member')}</TableHead>
          <TableHead>{t('organizationRole')}</TableHead>
          <TableHead className="w-32 text-right">{t('accessRoles')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {members.map((member) => (
          <TableRow key={member.id}>
            <TableCell>
              <div className="font-medium">{member.nickname || member.username}</div>
              <div className="text-xs text-muted-foreground">@{member.username}</div>
            </TableCell>
            <TableCell><Badge variant="secondary">{member.role}</Badge></TableCell>
            <TableCell className="text-right">
              <Button
                type="button"
                size="sm"
                variant="outline"
                icon={<UserCog className="size-4" aria-hidden="true" />}
                onClick={() => onManage(member)}
              >
                {canManage ? t('manage') : t('view')}
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function AccessRoleDialog({
  organizationId,
  role,
  catalog,
  open,
  onOpenChange,
}: {
  organizationId: number;
  role: AccessRole | null | undefined;
  catalog: string[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT('permission');
  const common = useT('common');
  const createMutation = useCreateAccessRole(organizationId);
  const updateMutation = useUpdateAccessRole(organizationId);
  const resetCreateMutation = createMutation.reset;
  const resetUpdateMutation = updateMutation.reset;
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [permissions, setPermissions] = useState<string[]>([]);
  const [clientError, setClientError] = useState<string>();
  const mutation = role ? updateMutation : createMutation;

  useEffect(() => {
    if (!open) return;
    setName(role?.name ?? '');
    setSlug(role?.slug ?? '');
    setPermissions(role?.permissions ?? []);
    setClientError(undefined);
    resetCreateMutation();
    resetUpdateMutation();
  }, [open, role, resetCreateMutation, resetUpdateMutation]);

  const submit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const options = {
      onSuccess: () => {
        toast.success(role ? t('roleUpdated') : t('roleCreated'));
        onOpenChange(false);
      },
    };
    if (role) {
      const parsed = updateAccessRoleSchema.safeParse({ name, permissions });
      if (!parsed.success) {
        setClientError(t('invalidRole'));
        return;
      }
      updateMutation.mutate({ roleId: role.id, input: parsed.data }, options);
    } else {
      const parsed = createAccessRoleSchema.safeParse({ name, slug, permissions });
      if (!parsed.success) {
        setClientError(t('invalidRole'));
        return;
      }
      createMutation.mutate(parsed.data, options);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{role ? t('editRole') : t('createRole')}</DialogTitle>
          <DialogDescription>{t('roleDialogDescription')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit} className="contents">
          <DialogBody className="space-y-5">
            {clientError || mutation.error ? (
              <Alert variant="destructive">
                <AlertDescription>{clientError ?? t('saveError')}</AlertDescription>
              </Alert>
            ) : null}
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="access-role-name">{t('name')}</Label>
                <Input
                  id="access-role-name"
                  required
                  minLength={2}
                  maxLength={100}
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="access-role-slug">{t('slug')}</Label>
                <Input
                  id="access-role-slug"
                  required
                  minLength={3}
                  maxLength={50}
                  pattern="[a-z0-9](?:[a-z0-9-]{1,48}[a-z0-9])"
                  value={slug}
                  disabled={Boolean(role)}
                  onChange={(event) => setSlug(event.target.value)}
                />
              </div>
            </div>
            <fieldset className="space-y-3">
              <legend className="text-sm font-medium">{t('permissions')}</legend>
              <div className="grid gap-2 sm:grid-cols-2">
                {catalog.map((permission) => {
                  const id = `access-role-permission-${permission}`;
                  return (
                    <label
                      key={permission}
                      htmlFor={id}
                      className="flex min-h-10 cursor-pointer items-center gap-3 rounded-md border px-3 py-2 hover:bg-muted/50"
                    >
                      <Checkbox
                        id={id}
                        checked={permissions.includes(permission)}
                        onCheckedChange={(checked) => {
                          setPermissions((current) => checked
                            ? [...current, permission].sort()
                            : current.filter((value) => value !== permission));
                        }}
                      />
                      <span className="min-w-0 break-all font-mono text-xs">{permission}</span>
                    </label>
                  );
                })}
              </div>
            </fieldset>
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {common('cancel')}
            </Button>
            <Button type="submit" loading={mutation.isPending}>{common('save')}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function DeleteAccessRoleDialog({
  organizationId,
  role,
  onOpenChange,
}: {
  organizationId: number;
  role?: AccessRole;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT('permission');
  const common = useT('common');
  const mutation = useDeleteAccessRole(organizationId);
  return (
    <Dialog open={Boolean(role)} onOpenChange={onOpenChange}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t('deleteRole')}</DialogTitle>
          <DialogDescription>{t('deleteRoleDescription', { name: role?.name ?? '' })}</DialogDescription>
        </DialogHeader>
        {mutation.error ? (
          <Alert variant="destructive"><AlertDescription>{t('deleteError')}</AlertDescription></Alert>
        ) : null}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {common('cancel')}
          </Button>
          <Button
            type="button"
            variant="destructive"
            loading={mutation.isPending}
            onClick={() => {
              if (!role) return;
              mutation.mutate(role.id, {
                onSuccess: () => {
                  toast.success(t('roleDeleted'));
                  onOpenChange(false);
                },
              });
            }}
          >
            {common('delete')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function MemberAccessRoleDialog({
  organizationId,
  member,
  roles,
  canManage,
  onOpenChange,
}: {
  organizationId: number;
  member?: OrganizationMember;
  roles: AccessRole[];
  canManage: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT('permission');
  const common = useT('common');
  const memberId = member?.id ?? 0;
  const query = useMemberAccessRoles(organizationId, memberId, Boolean(member));
  const mutation = useReplaceMemberAccessRoles(organizationId, memberId);
  const [selected, setSelected] = useState<number[]>([]);

  useEffect(() => {
    if (query.data) setSelected(query.data.access_role_ids);
  }, [query.data]);

  return (
    <Dialog open={Boolean(member)} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('memberRoleTitle', { name: member?.nickname || member?.username || '' })}</DialogTitle>
          <DialogDescription>{t('memberRoleDescription')}</DialogDescription>
        </DialogHeader>
        <DialogBody className="space-y-3">
          {query.error || mutation.error ? (
            <Alert variant="destructive"><AlertDescription>{t('assignmentError')}</AlertDescription></Alert>
          ) : query.isPending ? (
            <Skeleton className="h-24 w-full" />
          ) : roles.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('emptyRoles')}</p>
          ) : roles.map((role) => {
            const id = `member-${memberId}-role-${role.id}`;
            return (
              <label key={role.id} htmlFor={id} className="flex cursor-pointer items-start gap-3 rounded-md border p-3 hover:bg-muted/50">
                <Checkbox
                  id={id}
                  checked={selected.includes(role.id)}
                  disabled={!canManage}
                  onCheckedChange={(checked) => setSelected((current) => checked
                    ? [...current, role.id].sort((left, right) => left - right)
                    : current.filter((value) => value !== role.id))}
                />
                <span className="min-w-0">
                  <span className="block text-sm font-medium">{role.name}</span>
                  <span className="mt-1 block break-all font-mono text-xs text-muted-foreground">
                    {role.permissions.join(', ') || t('noPermissions')}
                  </span>
                </span>
              </label>
            );
          })}
        </DialogBody>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {canManage ? common('cancel') : t('close')}
          </Button>
          {canManage ? (
            <Button
              type="button"
              loading={mutation.isPending}
              disabled={query.isPending}
              onClick={() => mutation.mutate(
                { access_role_ids: selected },
                {
                  onSuccess: () => {
                    toast.success(t('assignmentSaved'));
                    onOpenChange(false);
                  },
                }
              )}
            >
              {common('save')}
            </Button>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function PermissionAlert({ message, onRetry }: { message: string; onRetry: () => void }) {
  const t = useT('permission');
  return (
    <Alert variant="destructive">
      <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
        <span>{message}</span>
        <Button type="button" size="sm" variant="outline" icon={<RefreshCw className="size-3.5" />} onClick={onRetry}>
          {t('retry')}
        </Button>
      </AlertDescription>
    </Alert>
  );
}

function PermissionManagementSkeleton() {
  return (
    <div className="space-y-4 py-6" aria-hidden="true">
      <Skeleton className="h-6 w-48" />
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-24 w-full" />
    </div>
  );
}

function hasPermission(context: PermissionContext | undefined, permission: string): boolean {
  return Boolean(context && (context.is_owner || context.permissions.includes(permission)));
}
