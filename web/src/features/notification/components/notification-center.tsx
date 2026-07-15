'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Bell,
  CheckCheck,
  Inbox,
  Loader2,
  Settings2,
} from 'lucide-react';
import { toast } from 'sonner';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Switch } from '@/components/ui/switch';
import {
  useMarkNotificationsRead,
  useNotificationPreference,
  useNotifications,
  useNotificationStatus,
  useReplaceNotificationPreference,
  useReplaceNotificationReadState,
} from '@/features/notification/hooks/use-notifications';
import type {
  NotificationItem,
  NotificationPreference,
} from '@/features/notification/types';
import { useT } from '@/i18n';
import { cn } from '@/utils';

export function NotificationCenter() {
  const router = useRouter();
  const t = useT('notification');
  const [open, setOpen] = useState(false);
  const [preferenceOpen, setPreferenceOpen] = useState(false);
  const [preferenceDraft, setPreferenceDraft] = useState<Partial<NotificationPreference>>({});
  const status = useNotificationStatus();
  const notifications = useNotifications(open);
  const preference = useNotificationPreference(preferenceOpen);
  const replaceReadState = useReplaceNotificationReadState();
  const markRead = useMarkNotificationsRead();
  const replacePreference = useReplaceNotificationPreference();
  const unreadCount = status.data?.unread_count ?? 0;
  const highestLoadedId = notifications.data?.items[0]?.id;

  async function openNotification(item: NotificationItem): Promise<void> {
    setOpen(false);
    if (!item.is_read) {
      try {
        await replaceReadState.mutateAsync({ notificationId: item.id, isRead: true });
      } catch {
        toast.error(t('readError'));
      }
    }
    if (item.action_url) router.push(item.action_url);
  }

  async function markAllRead(): Promise<void> {
    if (!highestLoadedId) return;
    try {
      await markRead.mutateAsync(highestLoadedId);
    } catch {
      toast.error(t('markAllError'));
    }
  }

  function showPreferences(): void {
    setOpen(false);
    setPreferenceDraft({});
    setPreferenceOpen(true);
  }

  async function savePreference(): Promise<void> {
    if (!preference.data) return;
    const input: NotificationPreference = {
      in_app_enabled:
        preferenceDraft.in_app_enabled ?? preference.data.in_app_enabled,
      email_enabled: preferenceDraft.email_enabled ?? preference.data.email_enabled,
    };
    try {
      await replacePreference.mutateAsync(input);
      toast.success(t('saved'));
      setPreferenceOpen(false);
      setPreferenceDraft({});
    } catch {
      toast.error(t('saveError'));
    }
  }

  return (
    <>
      <DropdownMenu open={open} onOpenChange={setOpen}>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" isIcon className="relative h-9 w-9 rounded-full">
            <Bell className="h-4 w-4 text-text-muted" />
            {unreadCount > 0 ? (
              <span
                aria-hidden="true"
                className="absolute right-1 top-1 flex min-w-4 items-center justify-center rounded-full bg-error px-1 text-[10px] font-semibold leading-4 text-white"
              >
                {unreadCount > 99 ? '99+' : unreadCount}
              </span>
            ) : null}
            <span className="sr-only">
              {unreadCount > 0 ? t('unreadCount', { count: unreadCount }) : t('open')}
            </span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="end"
          className="w-[calc(100vw-1rem)] max-w-[360px] overflow-hidden p-0"
        >
          <div className="flex h-12 items-center justify-between border-b px-3">
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold">{t('title')}</p>
              {unreadCount > 0 ? (
                <p className="text-xs text-text-muted">
                  {t('unreadCount', { count: unreadCount })}
                </p>
              ) : null}
            </div>
            {unreadCount > 0 && highestLoadedId ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="shrink-0"
                disabled={markRead.isPending}
                onClick={() => void markAllRead()}
              >
                {markRead.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <CheckCheck className="h-4 w-4" />
                )}
                {t('markAllRead')}
              </Button>
            ) : null}
          </div>

          <div className="max-h-[360px] overflow-y-auto overscroll-contain">
            {notifications.isPending ? (
              <NotificationMessage icon={Loader2} iconClassName="animate-spin" text={t('loading')} />
            ) : notifications.isError ? (
              <div className="flex min-h-36 flex-col items-center justify-center gap-3 px-6 text-center">
                <p className="text-sm text-text-muted">{t('loadError')}</p>
                <Button type="button" variant="outline" size="sm" onClick={() => notifications.refetch()}>
                  {t('retry')}
                </Button>
              </div>
            ) : notifications.data?.items.length ? (
              <div className="divide-y">
                {notifications.data.items.map(item => (
                  <button
                    key={item.id}
                    type="button"
                    className="relative flex min-h-20 w-full gap-3 px-3 py-3 text-left transition-colors hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                    onClick={() => void openNotification(item)}
                  >
                    <span
                      aria-hidden="true"
                      className={cn(
                        'mt-1.5 h-2 w-2 shrink-0 rounded-full',
                        item.is_read ? 'bg-border' : 'bg-primary'
                      )}
                    />
                    <span className="min-w-0 flex-1">
                      <span className={cn('block truncate text-sm', !item.is_read && 'font-semibold')}>
                        {item.title}
                      </span>
                      <span className="mt-0.5 line-clamp-2 block text-xs leading-5 text-text-muted">
                        {item.body}
                      </span>
                      <span className="mt-1 block text-[11px] text-text-subtle">
                        {formatNotificationDate(item.created_at)}
                      </span>
                    </span>
                    {!item.is_read ? <span className="sr-only">{t('unread')}</span> : null}
                  </button>
                ))}
              </div>
            ) : (
              <NotificationMessage icon={Inbox} text={t('empty')} />
            )}
          </div>

          <div className="border-t p-1.5">
            <Button
              type="button"
              variant="ghost"
              className="w-full justify-start"
              onClick={showPreferences}
            >
              <Settings2 className="h-4 w-4" />
              {t('preferences')}
            </Button>
          </div>
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog
        open={preferenceOpen}
        onOpenChange={nextOpen => {
          setPreferenceOpen(nextOpen);
          if (!nextOpen) setPreferenceDraft({});
        }}
      >
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>{t('preferences')}</DialogTitle>
            <DialogDescription>{t('preferencesDescription')}</DialogDescription>
          </DialogHeader>
          <DialogBody className="space-y-1">
            {preference.isPending ? (
              <NotificationMessage icon={Loader2} iconClassName="animate-spin" text={t('loading')} />
            ) : preference.isError ? (
              <div className="flex min-h-28 flex-col items-center justify-center gap-3 text-center">
                <p className="text-sm text-text-muted">{t('loadError')}</p>
                <Button type="button" variant="outline" size="sm" onClick={() => preference.refetch()}>
                  {t('retry')}
                </Button>
              </div>
            ) : preference.data ? (
              <>
                <PreferenceRow
                  label={t('inApp')}
                  description={t('inAppDescription')}
                  checked={preferenceDraft.in_app_enabled ?? preference.data.in_app_enabled}
                  disabled={replacePreference.isPending}
                  onCheckedChange={checked =>
                    setPreferenceDraft(current => ({ ...current, in_app_enabled: checked }))
                  }
                />
                <PreferenceRow
                  label={t('email')}
                  description={t('emailDescription')}
                  checked={preferenceDraft.email_enabled ?? preference.data.email_enabled}
                  disabled={replacePreference.isPending}
                  onCheckedChange={checked =>
                    setPreferenceDraft(current => ({ ...current, email_enabled: checked }))
                  }
                />
              </>
            ) : null}
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setPreferenceOpen(false)}>
              {t('cancel')}
            </Button>
            <Button
              type="button"
              disabled={!preference.data || replacePreference.isPending}
              onClick={() => void savePreference()}
            >
              {replacePreference.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              {t('save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function PreferenceRow({
  label,
  description,
  checked,
  disabled,
  onCheckedChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  disabled: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex min-h-20 cursor-pointer items-center gap-4 border-b py-3 last:border-b-0">
      <span className="min-w-0 flex-1">
        <span className="block text-sm font-medium">{label}</span>
        <span className="mt-1 block text-xs leading-5 text-text-muted">{description}</span>
      </span>
      <Switch
        aria-label={label}
        checked={checked}
        disabled={disabled}
        onCheckedChange={onCheckedChange}
      />
    </label>
  );
}

function NotificationMessage({
  icon: Icon,
  text,
  iconClassName,
}: {
  icon: typeof Inbox;
  text: string;
  iconClassName?: string;
}) {
  return (
    <div className="flex min-h-36 flex-col items-center justify-center gap-2 px-6 text-center text-text-muted">
      <Icon className={cn('h-5 w-5', iconClassName)} aria-hidden="true" />
      <p className="text-sm">{text}</p>
    </div>
  );
}

function formatNotificationDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}
