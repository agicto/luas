import { ListTree, RotateCw, Send, Webhook } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Pager } from '@/components/layout/pager';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogTitle,
} from '@/components/ui/dialog';
import { Skeleton } from '@/components/ui/skeleton';
import { StatusBadge } from '@/components/ui/status-badge';
import {
  useReplayWebhookDelivery,
  useWebhookAttempts,
  useWebhookDeliveries,
  useWebhookEndpoints,
} from '@/features/webhooks/hooks/use-webhooks';
import {
  isReplayable,
  type WebhookDelivery,
  type WebhookDeliveryStatus,
  webhookDeliveryStatuses,
} from '@/features/webhooks/types';
import { replayErrorKey } from '@/features/webhooks/webhook-errors';

const deliveryTone = {
  pending: 'neutral',
  processing: 'neutral',
  delivered: 'success',
  failed: 'danger',
  canceled: 'warning',
} as const;

function ListSkeleton() {
  return (
    <div className="flex flex-col gap-2 p-4" aria-busy="true">
      {Array.from({ length: 3 }, (_, index) => (
        <Skeleton key={index} className="h-9 w-full" />
      ))}
    </div>
  );
}

/** Secret-free webhook diagnosis and delivery replay for one organization. */
export function WebhookDiagnostics({ organizationId }: { organizationId: number }) {
  return (
    <>
      <WebhookEndpoints organizationId={organizationId} />
      <WebhookDeliveries organizationId={organizationId} />
    </>
  );
}

function WebhookEndpoints({ organizationId }: { organizationId: number }) {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const endpoints = useWebhookEndpoints(organizationId, page);
  const meta = endpoints.data?.meta;

  return (
    <section className="panel" aria-labelledby="webhook-endpoints-title">
      <div className="panel-header">
        <div className="flex items-start gap-3">
          <span className="icon-surface" aria-hidden="true">
            <Webhook className="size-4" />
          </span>
          <div>
            <h2 id="webhook-endpoints-title" className="panel-title">
              {t('webhooks.endpointsTitle')}
            </h2>
            <p className="panel-description">{t('webhooks.endpointsDescription')}</p>
          </div>
        </div>
      </div>
      <div className="panel-body overflow-x-auto p-0">
        {endpoints.isPending ? (
          <ListSkeleton />
        ) : endpoints.isError ? (
          <p role="alert" className="p-4 text-sm text-destructive">
            {t('webhooks.errors.unavailable')}
          </p>
        ) : endpoints.data.data.length === 0 ? (
          <p className="p-4 text-sm text-muted-foreground">{t('webhooks.endpointsEmpty')}</p>
        ) : (
          <table className="w-full min-w-[640px] text-sm">
            <thead className="border-b text-left text-xs text-muted-foreground">
              <tr>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t('webhooks.endpointColumns.endpoint')}
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t('webhooks.endpointColumns.status')}
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t('webhooks.endpointColumns.events')}
                </th>
              </tr>
            </thead>
            <tbody>
              {endpoints.data.data.map((endpoint) => (
                <tr key={endpoint.id} className="border-b align-top last:border-0">
                  <td className="px-4 py-2">
                    <div className="font-medium">
                      {endpoint.name} <span className="text-muted-foreground">#{endpoint.id}</span>
                    </div>
                    <div className="break-all text-xs text-muted-foreground">{endpoint.url}</div>
                  </td>
                  <td className="px-4 py-2">
                    <StatusBadge tone={endpoint.status === 'active' ? 'success' : 'warning'}>
                      {t(`webhooks.endpointStatus.${endpoint.status}`)}
                    </StatusBadge>
                    {endpoint.disabled_reason ? (
                      <div className="mt-1 text-xs text-muted-foreground">
                        <code>{endpoint.disabled_reason}</code>
                      </div>
                    ) : null}
                    {endpoint.consecutive_failures > 0 ? (
                      <div className="mt-1 text-xs text-muted-foreground">
                        {t('webhooks.consecutiveFailures', {
                          count: endpoint.consecutive_failures,
                        })}
                      </div>
                    ) : null}
                  </td>
                  <td className="px-4 py-2 text-xs">
                    {endpoint.event_types.map((eventType) => (
                      <code key={eventType} className="mr-2 inline-block">
                        {eventType}
                      </code>
                    ))}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      <Pager meta={meta} label={t('webhooks.endpointPagination')} onPageChange={setPage} />
    </section>
  );
}

type DeliveryDialog = { kind: 'attempts' | 'replay'; delivery: WebhookDelivery };

function WebhookDeliveries({ organizationId }: { organizationId: number }) {
  const { t, i18n } = useTranslation();
  const [status, setStatus] = useState<WebhookDeliveryStatus | undefined>(undefined);
  const [page, setPage] = useState(1);
  const [dialog, setDialog] = useState<DeliveryDialog | null>(null);
  const deliveries = useWebhookDeliveries(organizationId, { status, page });
  const attempts = useWebhookAttempts(
    organizationId,
    dialog?.kind === 'attempts' ? dialog.delivery.id : null,
  );
  const replay = useReplayWebhookDelivery(organizationId);
  const meta = deliveries.data?.meta;
  const dateFormat = new Intl.DateTimeFormat(i18n.language, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  });

  const closeDialog = () => {
    setDialog(null);
    replay.reset();
  };

  return (
    <section className="panel" aria-labelledby="webhook-deliveries-title">
      <div className="panel-header flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="flex items-start gap-3">
          <span className="icon-surface" aria-hidden="true">
            <Send className="size-4" />
          </span>
          <div>
            <h2 id="webhook-deliveries-title" className="panel-title">
              {t('webhooks.deliveriesTitle')}
            </h2>
            <p className="panel-description">
              {meta ? t('webhooks.deliveryTotal', { count: meta.total }) : t('common.loading')}
            </p>
          </div>
        </div>
        <label className="flex items-center gap-2">
          <span className="field-label">{t('webhooks.statusFilter')}</span>
          <select
            className="select-input"
            value={status ?? ''}
            onChange={(event) => {
              setStatus((event.target.value || undefined) as WebhookDeliveryStatus | undefined);
              setPage(1);
            }}
          >
            <option value="">{t('webhooks.allStatuses')}</option>
            {webhookDeliveryStatuses.map((value) => (
              <option key={value} value={value}>
                {t(`webhooks.deliveryStatus.${value}`)}
              </option>
            ))}
          </select>
        </label>
      </div>
      <div className="panel-body overflow-x-auto p-0">
        {deliveries.isPending ? (
          <ListSkeleton />
        ) : deliveries.isError ? (
          <p role="alert" className="p-4 text-sm text-destructive">
            {t('webhooks.errors.unavailable')}
          </p>
        ) : deliveries.data.data.length === 0 ? (
          <p className="p-4 text-sm text-muted-foreground">{t('webhooks.deliveriesEmpty')}</p>
        ) : (
          <table className="w-full min-w-[760px] text-sm">
            <thead className="border-b text-left text-xs text-muted-foreground">
              <tr>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t('webhooks.deliveryColumns.event')}
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t('webhooks.deliveryColumns.status')}
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t('webhooks.deliveryColumns.attempts')}
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t('webhooks.deliveryColumns.updated')}
                </th>
                <th scope="col" className="px-4 py-2 text-right font-medium">
                  {t('webhooks.deliveryColumns.actions')}
                </th>
              </tr>
            </thead>
            <tbody>
              {deliveries.data.data.map((delivery) => (
                <tr key={delivery.id} className="border-b align-top last:border-0">
                  <td className="px-4 py-2">
                    <div className="font-medium">{delivery.event_type}</div>
                    <div className="text-xs text-muted-foreground">
                      {delivery.message_id} ·{' '}
                      {t('webhooks.endpointRef', { id: delivery.endpoint_id })}
                    </div>
                  </td>
                  <td className="px-4 py-2">
                    <StatusBadge tone={deliveryTone[delivery.status]}>
                      {t(`webhooks.deliveryStatus.${delivery.status}`)}
                    </StatusBadge>
                    {delivery.failure_code || delivery.http_status ? (
                      <div className="mt-1 text-xs text-muted-foreground">
                        <code>
                          {[delivery.http_status, delivery.failure_code]
                            .filter(Boolean)
                            .join(' · ')}
                        </code>
                      </div>
                    ) : null}
                  </td>
                  <td className="px-4 py-2 tabular-nums">
                    {delivery.attempt_count}
                    {delivery.replay_count > 0 ? (
                      <span className="text-xs text-muted-foreground">
                        {' '}
                        {t('webhooks.replayed', { count: delivery.replay_count })}
                      </span>
                    ) : null}
                  </td>
                  <td className="whitespace-nowrap px-4 py-2 text-muted-foreground">
                    {dateFormat.format(new Date(delivery.updated_at))}
                  </td>
                  <td className="px-4 py-2">
                    <div className="flex justify-end gap-1">
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        onClick={() => setDialog({ kind: 'attempts', delivery })}
                      >
                        <ListTree aria-hidden="true" />
                        {t('webhooks.actions.attempts')}
                      </Button>
                      {isReplayable(delivery) ? (
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => setDialog({ kind: 'replay', delivery })}
                        >
                          <RotateCw aria-hidden="true" />
                          {t('webhooks.actions.replay')}
                        </Button>
                      ) : null}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      <Pager meta={meta} label={t('webhooks.deliveryPagination')} onPageChange={setPage} />

      <Dialog open={dialog !== null} onOpenChange={(open) => (open ? undefined : closeDialog())}>
        <DialogContent>
          {dialog?.kind === 'replay' ? (
            <>
              <DialogTitle>{t('webhooks.replayConfirm.title')}</DialogTitle>
              <DialogDescription>
                {t('webhooks.replayConfirm.description', { message: dialog.delivery.message_id })}
              </DialogDescription>
              {replay.isError ? (
                <p role="alert" className="text-sm text-destructive">
                  {t(replayErrorKey(replay.error))}
                </p>
              ) : null}
              <DialogFooter>
                <DialogClose asChild>
                  <Button type="button" variant="outline">
                    {t('common.cancel')}
                  </Button>
                </DialogClose>
                <Button
                  type="button"
                  disabled={replay.isPending}
                  onClick={() => replay.mutate(dialog.delivery.id, { onSuccess: closeDialog })}
                >
                  {t('webhooks.actions.replay')}
                </Button>
              </DialogFooter>
            </>
          ) : dialog ? (
            <>
              <DialogTitle>{t('webhooks.attemptsTitle')}</DialogTitle>
              <DialogDescription>
                {t('webhooks.attemptsDescription', { message: dialog.delivery.message_id })}
              </DialogDescription>
              {attempts.isPending ? (
                <Skeleton className="h-16 w-full" />
              ) : attempts.isError ? (
                <p role="alert" className="text-sm text-destructive">
                  {t('webhooks.errors.unavailable')}
                </p>
              ) : attempts.data.data.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t('webhooks.attemptsEmpty')}</p>
              ) : (
                <ol className="flex max-h-72 flex-col gap-2 overflow-y-auto text-sm">
                  {attempts.data.data.map((attempt) => (
                    <li key={attempt.id} className="rounded-md border px-3 py-2">
                      <div className="flex items-center justify-between gap-2">
                        <span className="font-medium">
                          {t('webhooks.attemptNumber', { number: attempt.number })}
                        </span>
                        <code className="text-xs">
                          {[attempt.outcome, attempt.http_status, attempt.failure_code]
                            .filter(Boolean)
                            .join(' · ')}
                        </code>
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {dateFormat.format(new Date(attempt.completed_at))} ·{' '}
                        {t('webhooks.duration', { ms: attempt.duration_ms })}
                      </div>
                    </li>
                  ))}
                </ol>
              )}
              <DialogFooter>
                <DialogClose asChild>
                  <Button type="button" variant="outline">
                    {t('common.close')}
                  </Button>
                </DialogClose>
              </DialogFooter>
            </>
          ) : null}
        </DialogContent>
      </Dialog>
    </section>
  );
}
