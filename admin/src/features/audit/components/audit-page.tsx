import { History, Search } from 'lucide-react';
import { type FormEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { StatusBadge } from '@/components/ui/status-badge';
import { ApiErrorCode } from '@/http/codes';
import { ApiError } from '@/http/client';
import { useAuditLogs } from '@/features/audit/hooks/use-audit-logs';
import type { AuditSearch } from '@/features/audit/types';

interface AuditPageProps {
  search: AuditSearch;
  onSearchChange: (search: AuditSearch) => void;
}

export function AuditPage({ search, onSearchChange }: AuditPageProps) {
  const { t, i18n } = useTranslation();
  const logs = useAuditLogs(search);
  const [action, setAction] = useState(search.action ?? '');
  const [userId, setUserId] = useState(search.user_id ? String(search.user_id) : '');
  const [from, setFrom] = useState(search.from ?? '');
  const [to, setTo] = useState(search.to ?? '');
  // Cursors of the pages before the current one, so Previous retraces the visited pages. A page
  // opened from a link has no trail, and Previous returns to the newest entries.
  const [trail, setTrail] = useState<string[]>([]);
  const meta = logs.data?.meta;
  const goTo = (cursor: string | undefined, nextTrail: string[]) => {
    setTrail(nextTrail);
    onSearchChange({ ...search, cursor });
  };
  const dateFormat = new Intl.DateTimeFormat(i18n.language, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  });
  const invalidRange =
    logs.error instanceof ApiError && logs.error.errorCode === ApiErrorCode.COMMON_INVALID_INPUT;

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const parsedUser = Number.parseInt(userId, 10);
    onSearchChange({
      action: action.trim() || undefined,
      user_id: Number.isInteger(parsedUser) && parsedUser > 0 ? parsedUser : undefined,
      from: from || undefined,
      to: to || undefined,
      cursor: undefined,
    });
    setTrail([]);
  };

  return (
    <div className="page-stack">
      <header className="page-header">
        <p className="page-eyebrow">{t('audit.eyebrow')}</p>
        <h1>{t('audit.title')}</h1>
        <p>{t('audit.description')}</p>
      </header>

      <section className="panel" aria-labelledby="audit-title">
        <div className="panel-header flex flex-col gap-3">
          <div className="flex items-start gap-3">
            <span className="icon-surface" aria-hidden="true">
              <History className="size-4" />
            </span>
            <div>
              <h2 id="audit-title" className="panel-title">
                {t('audit.listTitle')}
              </h2>
              <p className="panel-description">{t('audit.rangeHint')}</p>
            </div>
          </div>
          <form
            role="search"
            onSubmit={submit}
            className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-5"
          >
            <label className="flex flex-col gap-1">
              <span className="field-label">{t('audit.filters.action')}</span>
              <Input
                maxLength={120}
                value={action}
                onChange={(event) => setAction(event.target.value)}
              />
            </label>
            <label className="flex flex-col gap-1">
              <span className="field-label">{t('audit.filters.userId')}</span>
              <Input
                inputMode="numeric"
                pattern="[0-9]*"
                value={userId}
                onChange={(event) => setUserId(event.target.value)}
              />
            </label>
            <label className="flex flex-col gap-1">
              <span className="field-label">{t('audit.filters.from')}</span>
              <Input type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
            </label>
            <label className="flex flex-col gap-1">
              <span className="field-label">{t('audit.filters.to')}</span>
              <Input type="date" value={to} onChange={(event) => setTo(event.target.value)} />
            </label>
            <div className="flex items-end">
              <Button type="submit" className="w-full">
                <Search aria-hidden="true" />
                {t('audit.apply')}
              </Button>
            </div>
          </form>
        </div>

        <div className="panel-body overflow-x-auto p-0">
          {logs.isPending ? (
            <div className="flex flex-col gap-2 p-4" aria-busy="true">
              {Array.from({ length: 5 }, (_, index) => (
                <Skeleton key={index} className="h-8 w-full" />
              ))}
            </div>
          ) : logs.isError ? (
            <p role="alert" className="p-4 text-sm text-destructive">
              {invalidRange ? t('audit.errors.invalidRange') : t('audit.errors.unavailable')}
            </p>
          ) : logs.data.data.length === 0 ? (
            <p className="p-4 text-sm text-muted-foreground">{t('audit.empty')}</p>
          ) : (
            <table className="w-full min-w-[720px] text-sm">
              <thead className="border-b text-left text-xs text-muted-foreground">
                <tr>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('audit.columns.time')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('audit.columns.actor')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('audit.columns.action')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('audit.columns.request')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('audit.columns.status')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {logs.data.data.map((entry) => (
                  <tr key={entry.id} className="border-b align-top last:border-0">
                    <td className="whitespace-nowrap px-4 py-2 text-muted-foreground">
                      {dateFormat.format(new Date(entry.created_at))}
                    </td>
                    <td className="px-4 py-2">
                      {entry.user_id
                        ? t('audit.actorUser', { id: entry.user_id })
                        : t(`audit.actorType.${entry.actor_type}`, entry.actor_type)}
                    </td>
                    <td className="px-4 py-2">
                      <div className="font-medium">{entry.action}</div>
                      <div className="text-xs text-muted-foreground">{entry.resource}</div>
                    </td>
                    <td className="px-4 py-2">
                      <code className="text-xs">
                        {entry.method} {entry.path}
                      </code>
                      {entry.request_id ? (
                        <div className="text-xs text-muted-foreground">{entry.request_id}</div>
                      ) : null}
                    </td>
                    <td className="px-4 py-2">
                      <StatusBadge>{entry.status_code}</StatusBadge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {meta && (search.cursor || meta.has_more) ? (
          <nav
            className="flex items-center justify-end gap-2 border-t px-4 py-3 text-sm"
            aria-label={t('users.pagination')}
          >
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={!search.cursor}
              onClick={() => goTo(trail.at(-1) || undefined, trail.slice(0, -1))}
            >
              {t('users.previous')}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={!meta.next_cursor}
              onClick={() => goTo(meta.next_cursor ?? undefined, [...trail, search.cursor ?? ''])}
            >
              {t('users.next')}
            </Button>
          </nav>
        ) : null}
      </section>
    </div>
  );
}
