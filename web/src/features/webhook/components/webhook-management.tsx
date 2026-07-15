'use client';

import { useMemo, useState, type FormEvent } from 'react';
import { useLocale } from 'next-intl';
import {
  AlertCircle,
  ChevronLeft,
  ChevronRight,
  Copy,
  Eye,
  KeyRound,
  Pencil,
  Plus,
  RefreshCw,
  RotateCw,
  Send,
  Trash2,
  Webhook,
} from 'lucide-react';
import { toast } from 'sonner';

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
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
import { Switch } from '@/components/ui/switch';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  useCreateWebhookEndpoint,
  useDeleteWebhookEndpoint,
  useReplaceWebhookEndpointStatus,
  useRotateWebhookEndpointSecret,
  useTestWebhookEndpoint,
  useUpdateWebhookEndpoint,
  useWebhookAttempts,
  useWebhookDeliveries,
  useWebhookEndpoints,
  useWebhookEventTypes,
} from '@/features/webhook/hooks/use-webhooks';
import { webhookEndpointInputSchema } from '@/features/webhook/schemas';
import type {
  WebhookAttempt,
  WebhookDelivery,
  WebhookEndpoint,
  WebhookEndpointSecret,
} from '@/features/webhook/types';
import { resolveWebhookErrorKey } from '@/features/webhook/utils/webhook-error';
import { useT } from '@/i18n';

const webhookTestEvent = 'webhook.test' as const;

export function WebhookManagement({ organizationId }: { organizationId: number }) {
  const t = useT('webhook');
  const locale = useLocale();
  const [endpointPage, setEndpointPage] = useState(1);
  const [deliveryPage, setDeliveryPage] = useState(1);
  const [editor, setEditor] = useState<WebhookEndpoint | 'create' | null>(null);
  const [secret, setSecret] = useState<WebhookEndpointSecret | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<WebhookEndpoint | null>(null);
  const [rotateTarget, setRotateTarget] = useState<WebhookEndpoint | null>(null);
  const [attemptTarget, setAttemptTarget] = useState<WebhookDelivery | null>(null);
  const endpoints = useWebhookEndpoints(organizationId, endpointPage);
  const deliveries = useWebhookDeliveries(organizationId, deliveryPage);
  const eventTypes = useWebhookEventTypes(organizationId);
  const replaceStatus = useReplaceWebhookEndpointStatus(organizationId);
  const testEndpoint = useTestWebhookEndpoint(organizationId);
  const dateTime = useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }),
    [locale]
  );

  const changeStatus = (endpoint: WebhookEndpoint, enabled: boolean) => {
    replaceStatus.mutate(
      { endpoint, enabled },
      {
        onSuccess: () => toast.success(t(enabled ? 'messages.enabled' : 'messages.disabled')),
        onError: error => toast.error(t(resolveWebhookErrorKey(error))),
      }
    );
  };

  const queueTest = (endpoint: WebhookEndpoint) => {
    testEndpoint.mutate(
      { endpoint, idempotencyKey: `ui-${crypto.randomUUID()}` },
      {
        onSuccess: () => toast.success(t('messages.testQueued')),
        onError: error => toast.error(t(resolveWebhookErrorKey(error))),
      }
    );
  };

  return (
    <div className="space-y-10 py-6">
      <header className="flex flex-col gap-4 border-b pb-5 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <h2 className="flex items-center gap-2 text-base font-semibold">
            <Webhook className="size-4 text-muted-foreground" aria-hidden="true" />
            {t('title')}
          </h2>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">{t('description')}</p>
        </div>
        <Button
          type="button"
          size="sm"
          icon={<Plus className="size-4" />}
          disabled={eventTypes.isError}
          onClick={() => setEditor('create')}
        >
          {t('actions.create')}
        </Button>
      </header>

      <section aria-labelledby="webhook-endpoints-heading">
        <SectionHeading
          id="webhook-endpoints-heading"
          title={t('endpoints.title')}
          description={t('endpoints.description')}
          action={
            <IconAction
              label={t('actions.refresh')}
              icon={<RefreshCw className="size-4" />}
              disabled={endpoints.isFetching}
              onClick={() => void endpoints.refetch()}
            />
          }
        />
        {endpoints.isPending ? (
          <TableSkeleton />
        ) : endpoints.error || !endpoints.data ? (
          <LoadError
            message={t(resolveWebhookErrorKey(endpoints.error))}
            retryLabel={t('actions.retry')}
            onRetry={() => void endpoints.refetch()}
          />
        ) : endpoints.data.items.length === 0 ? (
          <EmptyState
            icon={<Webhook className="size-6" />}
            title={t('endpoints.emptyTitle')}
            description={t('endpoints.emptyDescription')}
          />
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('columns.endpoint')}</TableHead>
                  <TableHead>{t('columns.events')}</TableHead>
                  <TableHead>{t('columns.status')}</TableHead>
                  <TableHead>{t('columns.failures')}</TableHead>
                  <TableHead>{t('columns.updated')}</TableHead>
                  <TableHead className="w-48 text-right">{t('columns.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {endpoints.data.items.map(endpoint => (
                  <TableRow key={endpoint.id}>
                    <TableCell className="min-w-64 whitespace-normal py-3">
                      <div className="font-medium">{endpoint.name}</div>
                      <div
                        className="mt-1 max-w-80 truncate font-mono text-xs text-muted-foreground"
                        title={endpoint.url}
                      >
                        {endpoint.url}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {endpoint.event_types.map(eventType => (
                          <Badge key={eventType} variant="outline">
                            {eventType}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Switch
                          id={`webhook-endpoint-${endpoint.id}-status`}
                          checked={endpoint.status === 'active'}
                          disabled={replaceStatus.isPending}
                          aria-label={
                            endpoint.status === 'active'
                              ? t('endpointStatus.active')
                              : t('endpointStatus.disabled')
                          }
                          onCheckedChange={enabled => changeStatus(endpoint, enabled)}
                        />
                        <EndpointStatus status={endpoint.status} />
                      </div>
                    </TableCell>
                    <TableCell className="tabular-nums">{endpoint.consecutive_failures}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {dateTime.format(new Date(endpoint.updated_at))}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <IconAction
                          label={t('actions.test')}
                          icon={<Send className="size-4" />}
                          disabled={endpoint.status !== 'active' || testEndpoint.isPending}
                          onClick={() => queueTest(endpoint)}
                        />
                        <IconAction
                          label={t('actions.edit')}
                          icon={<Pencil className="size-4" />}
                          onClick={() => setEditor(endpoint)}
                        />
                        <IconAction
                          label={t('actions.rotate')}
                          icon={<RotateCw className="size-4" />}
                          onClick={() => setRotateTarget(endpoint)}
                        />
                        <IconAction
                          label={t('actions.delete')}
                          icon={<Trash2 className="size-4" />}
                          destructive
                          onClick={() => setDeleteTarget(endpoint)}
                        />
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pagination
              page={endpoints.data.meta.current_page}
              lastPage={endpoints.data.meta.last_page}
              onPage={setEndpointPage}
            />
          </>
        )}
      </section>

      <section aria-labelledby="webhook-deliveries-heading">
        <SectionHeading
          id="webhook-deliveries-heading"
          title={t('deliveries.title')}
          description={t('deliveries.description')}
          action={
            <IconAction
              label={t('actions.refresh')}
              icon={<RefreshCw className="size-4" />}
              disabled={deliveries.isFetching}
              onClick={() => void deliveries.refetch()}
            />
          }
        />
        {deliveries.isPending ? (
          <TableSkeleton />
        ) : deliveries.error || !deliveries.data ? (
          <LoadError
            message={t(resolveWebhookErrorKey(deliveries.error))}
            retryLabel={t('actions.retry')}
            onRetry={() => void deliveries.refetch()}
          />
        ) : deliveries.data.items.length === 0 ? (
          <EmptyState
            icon={<Send className="size-6" />}
            title={t('deliveries.emptyTitle')}
            description={t('deliveries.emptyDescription')}
          />
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('columns.message')}</TableHead>
                  <TableHead>{t('columns.endpoint')}</TableHead>
                  <TableHead>{t('columns.status')}</TableHead>
                  <TableHead>{t('columns.result')}</TableHead>
                  <TableHead>{t('columns.attempts')}</TableHead>
                  <TableHead>{t('columns.updated')}</TableHead>
                  <TableHead className="w-16 text-right">{t('columns.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {deliveries.data.items.map(delivery => (
                  <TableRow key={delivery.id}>
                    <TableCell className="min-w-56 whitespace-normal py-3">
                      <code className="text-xs">{delivery.message_id}</code>
                      <div className="mt-1 text-xs text-muted-foreground">
                        {delivery.event_type}
                      </div>
                    </TableCell>
                    <TableCell className="font-mono text-xs">#{delivery.endpoint_id}</TableCell>
                    <TableCell>
                      <DeliveryStatus status={delivery.status} />
                    </TableCell>
                    <TableCell className="max-w-64 whitespace-normal">
                      {delivery.http_status ? (
                        <span className="font-mono">HTTP {delivery.http_status}</span>
                      ) : delivery.failure_code ? (
                        <code className="break-all text-xs text-muted-foreground">
                          {delivery.failure_code}
                        </code>
                      ) : (
                        <span className="text-muted-foreground">-</span>
                      )}
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {delivery.attempt_count}
                      {delivery.replay_count > 0 ? ` / +${delivery.replay_count}` : ''}
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {dateTime.format(new Date(delivery.updated_at))}
                    </TableCell>
                    <TableCell className="text-right">
                      <IconAction
                        label={t('actions.attempts')}
                        icon={<Eye className="size-4" />}
                        onClick={() => setAttemptTarget(delivery)}
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pagination
              page={deliveries.data.meta.current_page}
              lastPage={deliveries.data.meta.last_page}
              onPage={setDeliveryPage}
            />
          </>
        )}
      </section>

      {editor !== null ? (
        <EndpointDialog
          key={editor === 'create' ? 'create' : `edit-${editor.id}-${editor.version}`}
          organizationId={organizationId}
          endpoint={editor === 'create' ? null : editor}
          eventTypes={eventTypes.data ?? [webhookTestEvent]}
          onClose={() => setEditor(null)}
          onSecret={value => {
            setSecret(value);
            setEditor(null);
          }}
        />
      ) : null}
      <SecretDialog secret={secret} dateTime={dateTime} onClose={() => setSecret(null)} />
      <DeleteDialog
        organizationId={organizationId}
        endpoint={deleteTarget}
        onClose={() => setDeleteTarget(null)}
      />
      <RotateDialog
        organizationId={organizationId}
        endpoint={rotateTarget}
        onClose={() => setRotateTarget(null)}
        onSecret={value => {
          setSecret(value);
          setRotateTarget(null);
        }}
      />
      {attemptTarget ? (
        <AttemptDialog
          organizationId={organizationId}
          delivery={attemptTarget}
          dateTime={dateTime}
          onClose={() => setAttemptTarget(null)}
        />
      ) : null}
    </div>
  );
}

function EndpointDialog({
  organizationId,
  endpoint,
  eventTypes,
  onClose,
  onSecret,
}: {
  organizationId: number;
  endpoint: WebhookEndpoint | null;
  eventTypes: readonly string[];
  onClose: () => void;
  onSecret: (secret: WebhookEndpointSecret) => void;
}) {
  const t = useT('webhook');
  const [name, setName] = useState(endpoint?.name ?? '');
  const [url, setUrl] = useState(endpoint?.url ?? '');
  const [testSelected, setTestSelected] = useState(
    endpoint?.event_types.includes(webhookTestEvent) ?? true
  );
  const [formError, setFormError] = useState<string>();
  const create = useCreateWebhookEndpoint(organizationId);
  const update = useUpdateWebhookEndpoint(organizationId);
  const mutationError = create.error ?? update.error;
  const pending = create.isPending || update.isPending;

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(undefined);
    const input = webhookEndpointInputSchema.safeParse({
      name: name.trim(),
      url: url.trim(),
      event_types: testSelected ? [webhookTestEvent] : [],
    });
    if (!input.success) {
      setFormError(t('form.invalid'));
      return;
    }
    try {
      if (endpoint) {
        await update.mutateAsync({ endpoint, input: input.data });
        toast.success(t('messages.updated'));
        onClose();
      } else {
        const result = await create.mutateAsync(input.data);
        create.reset();
        toast.success(t('messages.created'));
        onSecret(result);
      }
    } catch {
      // The localized mutation error remains visible in the dialog.
    }
  };

  return (
    <Dialog open onOpenChange={open => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t(endpoint ? 'form.editTitle' : 'form.createTitle')}</DialogTitle>
          <DialogDescription>{t('form.description')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={submit}>
          <DialogBody className="space-y-5">
            {formError || mutationError ? (
              <Alert variant="destructive">
                <AlertCircle className="size-4" aria-hidden="true" />
                <AlertDescription>
                  {formError ?? t(resolveWebhookErrorKey(mutationError))}
                </AlertDescription>
              </Alert>
            ) : null}
            <div className="space-y-2">
              <Label htmlFor="webhook-endpoint-name">{t('form.name')}</Label>
              <Input
                id="webhook-endpoint-name"
                value={name}
                maxLength={100}
                autoComplete="off"
                placeholder={t('form.namePlaceholder')}
                disabled={pending}
                onChange={event => setName(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="webhook-endpoint-url">{t('form.url')}</Label>
              <Input
                id="webhook-endpoint-url"
                type="url"
                inputMode="url"
                value={url}
                maxLength={2_048}
                autoComplete="url"
                spellCheck={false}
                placeholder={t('form.urlPlaceholder')}
                disabled={pending}
                onChange={event => setUrl(event.target.value)}
              />
            </div>
            <fieldset className="space-y-2">
              <legend className="text-sm font-medium">{t('form.eventTypes')}</legend>
              {eventTypes.map(eventType => (
                <div
                  key={eventType}
                  className="flex items-center gap-2 rounded-sm border px-3 py-2.5"
                >
                  <Checkbox
                    id={`webhook-event-${eventType}`}
                    checked={eventType === webhookTestEvent && testSelected}
                    disabled={eventType !== webhookTestEvent || pending}
                    onCheckedChange={checked => setTestSelected(checked === true)}
                  />
                  <Label htmlFor={`webhook-event-${eventType}`} className="font-mono text-xs">
                    {eventType}
                  </Label>
                </div>
              ))}
            </fieldset>
          </DialogBody>
          <DialogFooter className="mt-4">
            <Button type="button" variant="outline" disabled={pending} onClick={onClose}>
              {t('actions.cancel')}
            </Button>
            <Button type="submit" loading={pending}>
              {t(endpoint ? 'actions.save' : 'actions.create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function SecretDialog({
  secret,
  dateTime,
  onClose,
}: {
  secret: WebhookEndpointSecret | null;
  dateTime: Intl.DateTimeFormat;
  onClose: () => void;
}) {
  const t = useT('webhook');

  const copy = async () => {
    if (!secret) return;
    try {
      await navigator.clipboard.writeText(secret.signing_secret);
      toast.success(t('messages.copied'));
    } catch {
      toast.error(t('messages.copyFailed'));
    }
  };

  return (
    <Dialog open={secret !== null} onOpenChange={open => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('secret.title')}</DialogTitle>
          <DialogDescription>{t('secret.description')}</DialogDescription>
        </DialogHeader>
        {secret ? (
          <DialogBody className="space-y-4">
            <Alert>
              <KeyRound className="size-4" aria-hidden="true" />
              <AlertTitle>{t('secret.warningTitle')}</AlertTitle>
              <AlertDescription>{t('secret.warning')}</AlertDescription>
            </Alert>
            <div className="space-y-2">
              <Label htmlFor="webhook-signing-secret">{t('secret.label')}</Label>
              <div className="flex gap-2">
                <Input
                  id="webhook-signing-secret"
                  value={secret.signing_secret}
                  readOnly
                  autoComplete="off"
                  spellCheck={false}
                  className="font-mono text-xs"
                />
                <IconAction
                  label={t('actions.copy')}
                  icon={<Copy className="size-4" />}
                  bordered
                  onClick={() => void copy()}
                />
              </div>
            </div>
            {secret.previous_secret_expiry ? (
              <p className="text-sm text-muted-foreground">
                {t('secret.overlap', {
                  time: dateTime.format(new Date(secret.previous_secret_expiry)),
                })}
              </p>
            ) : null}
          </DialogBody>
        ) : null}
        <DialogFooter>
          <Button type="button" onClick={onClose}>
            {t('actions.done')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function DeleteDialog({
  organizationId,
  endpoint,
  onClose,
}: {
  organizationId: number;
  endpoint: WebhookEndpoint | null;
  onClose: () => void;
}) {
  const t = useT('webhook');
  const remove = useDeleteWebhookEndpoint(organizationId);

  const confirm = async () => {
    if (!endpoint) return;
    try {
      await remove.mutateAsync(endpoint);
      toast.success(t('messages.deleted'));
      onClose();
    } catch (error) {
      toast.error(t(resolveWebhookErrorKey(error)));
    }
  };

  return (
    <Dialog open={endpoint !== null} onOpenChange={open => !open && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t('delete.title')}</DialogTitle>
          <DialogDescription>
            {t('delete.description', { name: endpoint?.name ?? '' })}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" disabled={remove.isPending} onClick={onClose}>
            {t('actions.cancel')}
          </Button>
          <Button type="button" variant="destructive" loading={remove.isPending} onClick={confirm}>
            {t('actions.delete')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function RotateDialog({
  organizationId,
  endpoint,
  onClose,
  onSecret,
}: {
  organizationId: number;
  endpoint: WebhookEndpoint | null;
  onClose: () => void;
  onSecret: (secret: WebhookEndpointSecret) => void;
}) {
  const t = useT('webhook');
  const rotate = useRotateWebhookEndpointSecret(organizationId);

  const confirm = async () => {
    if (!endpoint) return;
    try {
      const result = await rotate.mutateAsync(endpoint);
      rotate.reset();
      toast.success(t('messages.rotated'));
      onSecret(result);
    } catch (error) {
      toast.error(t(resolveWebhookErrorKey(error)));
    }
  };

  return (
    <Dialog open={endpoint !== null} onOpenChange={open => !open && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t('rotate.title')}</DialogTitle>
          <DialogDescription>
            {t('rotate.description', { name: endpoint?.name ?? '' })}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" disabled={rotate.isPending} onClick={onClose}>
            {t('actions.cancel')}
          </Button>
          <Button type="button" loading={rotate.isPending} onClick={confirm}>
            {t('actions.rotate')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function AttemptDialog({
  organizationId,
  delivery,
  dateTime,
  onClose,
}: {
  organizationId: number;
  delivery: WebhookDelivery;
  dateTime: Intl.DateTimeFormat;
  onClose: () => void;
}) {
  const t = useT('webhook');
  const attempts = useWebhookAttempts(organizationId, delivery.id);

  return (
    <Dialog open onOpenChange={open => !open && onClose()}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t('attempts.title')}</DialogTitle>
          <DialogDescription>
            {t('attempts.description', { messageId: delivery.message_id })}
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          {attempts.isPending ? (
            <TableSkeleton />
          ) : attempts.error || !attempts.data ? (
            <LoadError
              message={t(resolveWebhookErrorKey(attempts.error))}
              retryLabel={t('actions.retry')}
              onRetry={() => void attempts.refetch()}
            />
          ) : attempts.data.items.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">{t('attempts.empty')}</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('columns.number')}</TableHead>
                  <TableHead>{t('columns.result')}</TableHead>
                  <TableHead>{t('columns.duration')}</TableHead>
                  <TableHead>{t('columns.completed')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {attempts.data.items.map(attempt => (
                  <TableRow key={attempt.id}>
                    <TableCell className="tabular-nums">{attempt.number}</TableCell>
                    <TableCell>
                      <AttemptOutcome attempt={attempt} />
                    </TableCell>
                    <TableCell className="tabular-nums">{attempt.duration_ms} ms</TableCell>
                    <TableCell className="text-muted-foreground">
                      {dateTime.format(new Date(attempt.completed_at))}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </DialogBody>
        <DialogFooter>
          <Button type="button" onClick={onClose}>
            {t('actions.done')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SectionHeading({
  id,
  title,
  description,
  action,
}: {
  id: string;
  title: string;
  description: string;
  action: React.ReactNode;
}) {
  return (
    <div className="mb-4 flex items-start justify-between gap-4">
      <div>
        <h3 id={id} className="text-sm font-semibold">
          {title}
        </h3>
        <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      </div>
      {action}
    </div>
  );
}

function IconAction({
  label,
  icon,
  bordered = false,
  destructive = false,
  disabled = false,
  onClick,
}: {
  label: string;
  icon: React.ReactNode;
  bordered?: boolean;
  destructive?: boolean;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          isIcon
          size="sm"
          variant={destructive ? 'destructive' : bordered ? 'outline' : 'ghost'}
          aria-label={label}
          disabled={disabled}
          onClick={onClick}
        >
          {icon}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

function Pagination({
  page,
  lastPage,
  onPage,
}: {
  page: number;
  lastPage: number;
  onPage: (page: number) => void;
}) {
  const t = useT('webhook');
  return (
    <nav
      className="mt-4 flex items-center justify-end gap-2"
      aria-label={t('messages.page', { page, lastPage })}
    >
      <IconAction
        label={t('actions.previous')}
        icon={<ChevronLeft className="size-4" />}
        bordered
        disabled={page <= 1}
        onClick={() => onPage(page - 1)}
      />
      <span className="min-w-28 text-center text-xs text-muted-foreground">
        {t('messages.page', { page, lastPage })}
      </span>
      <IconAction
        label={t('actions.next')}
        icon={<ChevronRight className="size-4" />}
        bordered
        disabled={page >= lastPage}
        onClick={() => onPage(page + 1)}
      />
    </nav>
  );
}

function EndpointStatus({ status }: { status: WebhookEndpoint['status'] }) {
  const t = useT('webhook');
  return (
    <Badge variant={status === 'active' ? 'outline' : 'secondary'}>
      {t(status === 'active' ? 'endpointStatus.active' : 'endpointStatus.disabled')}
    </Badge>
  );
}

function DeliveryStatus({ status }: { status: WebhookDelivery['status'] }) {
  const t = useT('webhook');
  const labels = {
    pending: 'deliveryStatus.pending',
    processing: 'deliveryStatus.processing',
    delivered: 'deliveryStatus.delivered',
    failed: 'deliveryStatus.failed',
    canceled: 'deliveryStatus.canceled',
  } as const;
  return (
    <Badge
      variant={
        status === 'failed' ? 'destructive' : status === 'delivered' ? 'outline' : 'secondary'
      }
    >
      {t(labels[status])}
    </Badge>
  );
}

function AttemptOutcome({ attempt }: { attempt: WebhookAttempt }) {
  const t = useT('webhook');
  const label =
    attempt.outcome === 'retry_scheduled'
      ? t('outcome.retryScheduled')
      : attempt.outcome === 'delivered'
        ? t('outcome.delivered')
        : t('outcome.failed');
  return (
    <div className="space-y-1">
      <Badge variant={attempt.outcome === 'failed' ? 'destructive' : 'outline'}>{label}</Badge>
      {attempt.http_status ? (
        <div className="font-mono text-xs">HTTP {attempt.http_status}</div>
      ) : null}
      {attempt.failure_code ? (
        <code className="block text-xs text-muted-foreground">{attempt.failure_code}</code>
      ) : null}
    </div>
  );
}

function LoadError({
  message,
  retryLabel,
  onRetry,
}: {
  message: string;
  retryLabel: string;
  onRetry: () => void;
}) {
  return (
    <Alert variant="destructive">
      <AlertCircle className="size-4" aria-hidden="true" />
      <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
        <span>{message}</span>
        <Button
          type="button"
          size="sm"
          variant="outline"
          icon={<RefreshCw className="size-3.5" />}
          onClick={onRetry}
        >
          {retryLabel}
        </Button>
      </AlertDescription>
    </Alert>
  );
}

function EmptyState({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="flex min-h-36 flex-col items-center justify-center gap-3 border border-dashed p-6 text-center">
      <div className="text-muted-foreground" aria-hidden="true">
        {icon}
      </div>
      <div>
        <p className="text-sm font-medium">{title}</p>
        <p className="mt-1 text-sm text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}

function TableSkeleton() {
  return (
    <div className="space-y-3" aria-hidden="true">
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-14 w-full" />
      <Skeleton className="h-14 w-full" />
    </div>
  );
}
