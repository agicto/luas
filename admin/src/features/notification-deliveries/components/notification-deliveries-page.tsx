import { BellRing, Search } from 'lucide-react';
import { type FormEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Pager } from '@/components/layout/pager';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { StatusBadge } from '@/components/ui/status-badge';
import { useNotificationDeliveries } from '@/features/notification-deliveries/hooks/use-notification-deliveries';
import {
  type NotificationChannel,
  type NotificationDeliverySearch,
  type NotificationDeliveryStatus,
  notificationChannels,
  notificationDeliveryStatuses,
} from '@/features/notification-deliveries/types';

interface NotificationDeliveriesPageProps {
  search: NotificationDeliverySearch;
  onSearchChange: (search: NotificationDeliverySearch) => void;
}

const statusTone = {
  pending: 'neutral',
  processing: 'neutral',
  delivered: 'success',
  failed: 'danger',
} as const;

export function NotificationDeliveriesPage({
  search,
  onSearchChange,
}: NotificationDeliveriesPageProps) {
  const { t, i18n } = useTranslation();
  const deliveries = useNotificationDeliveries(search);
  const [status, setStatus] = useState<string>(search.status ?? '');
  const [channel, setChannel] = useState<string>(search.channel ?? '');
  const [userId, setUserId] = useState(search.user_id ? String(search.user_id) : '');
  const meta = deliveries.data?.meta;
  const dateFormat = new Intl.DateTimeFormat(i18n.language, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  });

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const parsedUser = Number.parseInt(userId, 10);
    onSearchChange({
      status: (status || undefined) as NotificationDeliveryStatus | undefined,
      channel: (channel || undefined) as NotificationChannel | undefined,
      user_id: Number.isInteger(parsedUser) && parsedUser > 0 ? parsedUser : undefined,
      page: undefined,
    });
  };

  return (
    <div className="page-stack">
      <header className="page-header">
        <p className="page-eyebrow">{t('notificationDeliveries.eyebrow')}</p>
        <h1>{t('notificationDeliveries.title')}</h1>
        <p>{t('notificationDeliveries.description')}</p>
      </header>

      <section className="panel" aria-labelledby="notification-deliveries-title">
        <div className="panel-header flex flex-col gap-3">
          <div className="flex items-start gap-3">
            <span className="icon-surface" aria-hidden="true">
              <BellRing className="size-4" />
            </span>
            <div>
              <h2 id="notification-deliveries-title" className="panel-title">
                {t('notificationDeliveries.listTitle')}
              </h2>
              <p className="panel-description">
                {meta
                  ? t('notificationDeliveries.total', { count: meta.total })
                  : t('notificationDeliveries.privacyHint')}
              </p>
            </div>
          </div>
          <form
            role="search"
            onSubmit={submit}
            className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4"
          >
            <label className="flex flex-col gap-1">
              <span className="field-label">{t('notificationDeliveries.filters.status')}</span>
              <select
                className="select-input"
                value={status}
                onChange={(event) => setStatus(event.target.value)}
              >
                <option value="">{t('notificationDeliveries.any')}</option>
                {notificationDeliveryStatuses.map((value) => (
                  <option key={value} value={value}>
                    {t(`notificationDeliveries.status.${value}`)}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col gap-1">
              <span className="field-label">{t('notificationDeliveries.filters.channel')}</span>
              <select
                className="select-input"
                value={channel}
                onChange={(event) => setChannel(event.target.value)}
              >
                <option value="">{t('notificationDeliveries.any')}</option>
                {notificationChannels.map((value) => (
                  <option key={value} value={value}>
                    {t(`notificationDeliveries.channel.${value}`)}
                  </option>
                ))}
              </select>
            </label>
            <label className="flex flex-col gap-1">
              <span className="field-label">{t('notificationDeliveries.filters.userId')}</span>
              <Input
                inputMode="numeric"
                pattern="[0-9]*"
                value={userId}
                onChange={(event) => setUserId(event.target.value)}
              />
            </label>
            <div className="flex items-end">
              <Button type="submit" className="w-full">
                <Search aria-hidden="true" />
                {t('notificationDeliveries.apply')}
              </Button>
            </div>
          </form>
        </div>

        <div className="panel-body overflow-x-auto p-0">
          {deliveries.isPending ? (
            <div className="flex flex-col gap-2 p-4" aria-busy="true">
              {Array.from({ length: 5 }, (_, index) => (
                <Skeleton key={index} className="h-8 w-full" />
              ))}
            </div>
          ) : deliveries.isError ? (
            <p role="alert" className="p-4 text-sm text-destructive">
              {t('notificationDeliveries.errors.unavailable')}
            </p>
          ) : deliveries.data.data.length === 0 ? (
            <p className="p-4 text-sm text-muted-foreground">{t('notificationDeliveries.empty')}</p>
          ) : (
            <table className="w-full min-w-[720px] text-sm">
              <thead className="border-b text-left text-xs text-muted-foreground">
                <tr>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('notificationDeliveries.columns.notification')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('notificationDeliveries.columns.recipient')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('notificationDeliveries.columns.channel')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('notificationDeliveries.columns.status')}
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    {t('notificationDeliveries.columns.updated')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {deliveries.data.data.map((delivery) => (
                  <tr key={delivery.id} className="border-b align-top last:border-0">
                    <td className="px-4 py-2">
                      <div className="font-medium">{delivery.kind}</div>
                      <div className="text-xs text-muted-foreground">
                        {t('notificationDeliveries.notificationRef', {
                          id: delivery.notification_id,
                        })}
                      </div>
                    </td>
                    <td className="px-4 py-2">
                      {t('notificationDeliveries.userRef', { id: delivery.user_id })}
                    </td>
                    <td className="px-4 py-2">
                      {t(`notificationDeliveries.channel.${delivery.channel}`)}
                    </td>
                    <td className="px-4 py-2">
                      <StatusBadge tone={statusTone[delivery.status]}>
                        {t(`notificationDeliveries.status.${delivery.status}`)}
                      </StatusBadge>
                      <div className="mt-1 text-xs text-muted-foreground">
                        {t('notificationDeliveries.attempts', { count: delivery.attempts })}
                        {delivery.last_failure_code ? (
                          <>
                            {' · '}
                            <code>{delivery.last_failure_code}</code>
                          </>
                        ) : null}
                      </div>
                    </td>
                    <td className="whitespace-nowrap px-4 py-2 text-muted-foreground">
                      {dateFormat.format(new Date(delivery.updated_at))}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        <Pager
          meta={meta}
          label={t('notificationDeliveries.pagination')}
          onPageChange={(page) => onSearchChange({ ...search, page })}
        />
      </section>
    </div>
  );
}
