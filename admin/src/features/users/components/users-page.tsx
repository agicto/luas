import { Ban, CircleCheck, LogOut, Search, Users } from 'lucide-react';
import { type FormEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { StatusBadge } from '@/components/ui/status-badge';
import { useUserAction, useUsers } from '@/features/users/hooks/use-users';
import {
  type ManagedUser,
  type UserAction,
  type UserSearch,
  type UserStatusFilter,
  userStatusFilters,
} from '@/features/users/types';
import { userActionErrorKey } from '@/features/users/user-action-errors';
import { cn } from '@/lib/cn';

interface UsersPageProps {
  search: UserSearch;
  onSearchChange: (search: UserSearch) => void;
}

interface PendingAction {
  action: UserAction;
  user: ManagedUser;
}

export function UsersPage({ search, onSearchChange }: UsersPageProps) {
  const { t, i18n } = useTranslation();
  const users = useUsers(search);
  const userAction = useUserAction();
  const [query, setQuery] = useState(search.q ?? '');
  const [pending, setPending] = useState<PendingAction | null>(null);
  const status = search.status ?? 'all';
  const page = users.data?.meta;
  const dateFormat = new Intl.DateTimeFormat(i18n.language, {
    dateStyle: 'medium',
    timeStyle: 'short',
  });

  const submitSearch = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onSearchChange({ ...search, q: query.trim() || undefined, page: undefined });
  };

  const confirm = () => {
    if (!pending) {
      return;
    }
    userAction.mutate(
      { action: pending.action, id: pending.user.id },
      { onSuccess: () => setPending(null) },
    );
  };

  return (
    <div className="page-stack">
      <header className="page-header">
        <p className="page-eyebrow">{t('users.eyebrow')}</p>
        <h1>{t('users.title')}</h1>
        <p>{t('users.description')}</p>
      </header>

      <section className="panel" aria-labelledby="users-title">
        <div className="panel-header flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div className="flex items-start gap-3">
            <span className="icon-surface" aria-hidden="true">
              <Users className="size-4" />
            </span>
            <div>
              <h2 id="users-title" className="panel-title">
                {t('users.listTitle')}
              </h2>
              <p className="panel-description">
                {page ? t('users.total', { count: page.total }) : t('users.loading')}
              </p>
            </div>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <form role="search" onSubmit={submitSearch} className="flex gap-2">
              <Input
                aria-label={t('users.search')}
                placeholder={t('users.searchPlaceholder')}
                maxLength={100}
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                className="w-full sm:w-56"
              />
              <Button type="submit" size="icon" variant="outline" aria-label={t('users.search')}>
                <Search aria-hidden="true" />
              </Button>
            </form>
            <div className="segmented-control" role="group" aria-label={t('users.statusFilter')}>
              {userStatusFilters.map((value: UserStatusFilter) => (
                <Button
                  key={value}
                  type="button"
                  size="sm"
                  variant="ghost"
                  aria-pressed={status === value}
                  className={cn('flex-1', status === value && 'bg-background shadow-sm')}
                  onClick={() => onSearchChange({ ...search, status: value, page: undefined })}
                >
                  {t(`users.status.${value}`)}
                </Button>
              ))}
            </div>
          </div>
        </div>

        <div className="panel-body overflow-x-auto p-0">
          {users.isPending ? (
            <div className="flex flex-col gap-2 p-4" aria-busy="true">
              {Array.from({ length: 4 }, (_, index) => (
                <Skeleton key={index} className="h-9 w-full" />
              ))}
            </div>
          ) : users.isError ? (
            <p role="alert" className="p-4 text-sm text-destructive">
              {t('users.errors.unavailable')}
            </p>
          ) : users.data.data.length === 0 ? (
            <p className="p-4 text-sm text-muted-foreground">{t('users.empty')}</p>
          ) : (
            <table className="w-full min-w-[640px] text-sm">
              <thead className="border-b text-left text-xs text-muted-foreground">
                <tr>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('users.columns.account')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('users.columns.status')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('users.columns.lastLogin')}
                  </th>
                  <th scope="col" className="px-4 py-2 text-right font-medium">
                    {t('users.columns.actions')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {users.data.data.map((user) => (
                  <tr key={user.id} className="border-b last:border-0">
                    <td className="px-4 py-2">
                      <div className="font-medium">{user.nickname || user.username}</div>
                      <div className="text-xs text-muted-foreground">
                        {user.username} · {user.email}
                      </div>
                    </td>
                    <td className="px-4 py-2">
                      <div className="flex flex-wrap gap-1">
                        <StatusBadge>{t(`users.status.${user.status}`)}</StatusBadge>
                        {user.is_operator ? <StatusBadge>{t('users.operator')}</StatusBadge> : null}
                      </div>
                    </td>
                    <td className="px-4 py-2 text-muted-foreground">
                      {user.last_login
                        ? dateFormat.format(new Date(user.last_login))
                        : t('users.never')}
                    </td>
                    <td className="px-4 py-2">
                      {user.is_operator ? (
                        <p className="text-right text-xs text-muted-foreground">
                          {t('users.protectedHint')}
                        </p>
                      ) : (
                        <div className="flex justify-end gap-1">
                          {user.status === 'active' ? (
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              onClick={() => setPending({ action: 'disable', user })}
                            >
                              <Ban aria-hidden="true" />
                              {t('users.actions.disable')}
                            </Button>
                          ) : (
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              onClick={() => setPending({ action: 'enable', user })}
                            >
                              <CircleCheck aria-hidden="true" />
                              {t('users.actions.enable')}
                            </Button>
                          )}
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            onClick={() => setPending({ action: 'revokeSessions', user })}
                          >
                            <LogOut aria-hidden="true" />
                            {t('users.actions.revokeSessions')}
                          </Button>
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {page && page.last_page > 1 ? (
          <nav
            className="flex items-center justify-between border-t px-4 py-3 text-sm"
            aria-label={t('users.pagination')}
          >
            <span className="text-muted-foreground">
              {t('users.page', { page: page.current_page, pages: page.last_page })}
            </span>
            <div className="flex gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={page.current_page <= 1}
                onClick={() => onSearchChange({ ...search, page: page.current_page - 1 })}
              >
                {t('users.previous')}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={page.current_page >= page.last_page}
                onClick={() => onSearchChange({ ...search, page: page.current_page + 1 })}
              >
                {t('users.next')}
              </Button>
            </div>
          </nav>
        ) : null}
      </section>

      <Dialog
        open={pending !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPending(null);
            userAction.reset();
          }
        }}
      >
        <DialogContent>
          {pending ? (
            <>
              <DialogTitle>{t(`users.confirm.${pending.action}.title`)}</DialogTitle>
              <DialogDescription>
                {t(`users.confirm.${pending.action}.description`, {
                  name: pending.user.nickname || pending.user.username,
                })}
              </DialogDescription>
              {userAction.isError ? (
                <p role="alert" className="text-sm text-destructive">
                  {t(userActionErrorKey(userAction.error))}
                </p>
              ) : null}
              <DialogFooter>
                <DialogClose asChild>
                  <Button type="button" variant="outline">
                    {t('users.cancel')}
                  </Button>
                </DialogClose>
                <Button
                  type="button"
                  variant={pending.action === 'enable' ? 'default' : 'destructive'}
                  disabled={userAction.isPending}
                  onClick={confirm}
                >
                  {t(`users.actions.${pending.action}`)}
                </Button>
              </DialogFooter>
            </>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
