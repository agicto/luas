import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { NextIntlClientProvider, type AbstractIntlMessages } from 'next-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { NotificationCenter } from '@/features/notification/components/notification-center';
import { messages } from '@/i18n/modules';

const routerPush = vi.hoisted(() => vi.fn());
const state = vi.hoisted(() => ({
  status: { data: { unread_count: 1 } },
  notifications: {
    data: undefined as unknown,
    isPending: false,
    isError: false,
    refetch: vi.fn(),
  },
  preference: {
    data: { in_app_enabled: true, email_enabled: true },
    isPending: false,
    isError: false,
    refetch: vi.fn(),
  },
  replaceReadState: { isPending: false, mutateAsync: vi.fn() },
  markRead: { isPending: false, mutateAsync: vi.fn() },
  replacePreference: { isPending: false, mutateAsync: vi.fn() },
}));

vi.mock('next/navigation', () => ({ useRouter: () => ({ push: routerPush }) }));
vi.mock('sonner', () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock('@/features/notification/hooks/use-notifications', () => ({
  useNotificationStatus: () => state.status,
  useNotifications: () => state.notifications,
  useNotificationPreference: () => state.preference,
  useReplaceNotificationReadState: () => state.replaceReadState,
  useMarkNotificationsRead: () => state.markRead,
  useReplaceNotificationPreference: () => state.replacePreference,
}));

const items = [
  {
    id: 42,
    kind: 'billing.invoice_paid',
    title: '<b>Invoice paid</b>',
    body: '<script>alert(1)</script>',
    action_url: '/console/invoices/1042',
    is_read: false,
    read_at: null,
    created_at: '2026-07-15T14:00:00Z',
  },
  {
    id: 41,
    kind: 'starter.workspace_ready',
    title: 'Workspace ready',
    body: 'The workspace is ready.',
    is_read: true,
    read_at: '2026-07-15T13:00:00Z',
    created_at: '2026-07-15T13:00:00Z',
  },
];

describe('notification center', () => {
  beforeEach(() => {
    state.status.data = { unread_count: 1 };
    state.notifications.data = { items, meta: { total: 2 } };
    state.notifications.isPending = false;
    state.notifications.isError = false;
    state.notifications.refetch.mockReset();
    state.preference.data = { in_app_enabled: true, email_enabled: true };
    state.preference.isPending = false;
    state.preference.isError = false;
    state.preference.refetch.mockReset();
    state.replaceReadState.isPending = false;
    state.replaceReadState.mutateAsync.mockReset();
    state.replaceReadState.mutateAsync.mockResolvedValue(items[0]);
    state.markRead.isPending = false;
    state.markRead.mutateAsync.mockReset();
    state.markRead.mutateAsync.mockResolvedValue({ updated_count: 1, unread_count: 0 });
    state.replacePreference.isPending = false;
    state.replacePreference.mutateAsync.mockReset();
    state.replacePreference.mutateAsync.mockResolvedValue({
      in_app_enabled: true,
      email_enabled: false,
    });
    routerPush.mockReset();
  });

  it('renders provider content as text and opens only a safe local action', async () => {
    renderWithMessages(<NotificationCenter />);
    fireEvent.pointerDown(screen.getByRole('button', { name: '1 unread notifications' }));

    expect(await screen.findByText('<b>Invoice paid</b>')).toBeInTheDocument();
    expect(screen.getByText('<script>alert(1)</script>')).toBeInTheDocument();
    expect(document.querySelector('script')).toBeNull();
    fireEvent.click(screen.getByText('<b>Invoice paid</b>'));

    await waitFor(() => {
      expect(state.replaceReadState.mutateAsync).toHaveBeenCalledWith({
        notificationId: 42,
        isRead: true,
      });
      expect(routerPush).toHaveBeenCalledWith('/console/invoices/1042');
    });
  });

  it('uses the newest loaded id as a race-safe mark-all high-water mark', async () => {
    renderWithMessages(<NotificationCenter />);
    fireEvent.pointerDown(screen.getByRole('button', { name: '1 unread notifications' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Mark all as read' }));

    await waitFor(() => expect(state.markRead.mutateAsync).toHaveBeenCalledWith(42));
  });

  it('loads and replaces both channel preferences explicitly', async () => {
    renderWithMessages(<NotificationCenter />);
    fireEvent.pointerDown(screen.getByRole('button', { name: '1 unread notifications' }));
    fireEvent.click(await screen.findByRole('button', { name: 'Notification preferences' }));

    const emailSwitch = await screen.findByRole('switch', { name: 'Email notifications' });
    fireEvent.click(emailSwitch);
    fireEvent.click(screen.getByRole('button', { name: 'Save preferences' }));

    await waitFor(() => {
      expect(state.replacePreference.mutateAsync).toHaveBeenCalledWith({
        in_app_enabled: true,
        email_enabled: false,
      });
    });
  });
});

function renderWithMessages(children: React.ReactNode) {
  return render(
    <NextIntlClientProvider locale="en-US" messages={messages as unknown as AbstractIntlMessages}>
      {children}
    </NextIntlClientProvider>
  );
}
