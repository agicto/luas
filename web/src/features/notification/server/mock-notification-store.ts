import 'server-only';

import type { AuthUser } from '@/features/auth/types';
import type {
  NotificationFilter,
  NotificationItem,
  NotificationPage,
  NotificationPreference,
  NotificationReadStateResult,
  NotificationStatus,
} from '@/features/notification/types';

interface MockNotificationState {
  items: NotificationItem[];
  preference: NotificationPreference;
}

let nextNotificationId = 1;
const states = new Map<string, MockNotificationState>();

export const mockNotificationStore = {
  list(
    user: AuthUser,
    page: number,
    perPage: number,
    status: NotificationFilter
  ): NotificationPage {
    const state = stateFor(user);
    const matching = state.items
      .filter(item => status === 'all' || !item.is_read)
      .sort((left, right) => right.id - left.id);
    const start = (page - 1) * perPage;
    const items = matching.slice(start, start + perPage).map(cloneNotification);
    return {
      items,
      meta: pageMeta(matching.length, page, perPage, items.length),
      links: pageLinks(matching.length, page, perPage, status),
    };
  },

  status(user: AuthUser): NotificationStatus {
    return {
      unread_count: stateFor(user).items.filter(item => !item.is_read).length,
    };
  },

  replaceReadState(
    user: AuthUser,
    notificationId: number,
    isRead: boolean
  ): NotificationItem | null {
    const item = stateFor(user).items.find(candidate => candidate.id === notificationId);
    if (!item) return null;
    item.is_read = isRead;
    item.read_at = isRead ? new Date().toISOString() : null;
    return cloneNotification(item);
  },

  markReadThrough(user: AuthUser, throughId: number): NotificationReadStateResult {
    const state = stateFor(user);
    let updatedCount = 0;
    const readAt = new Date().toISOString();
    for (const item of state.items) {
      if (item.id <= throughId && !item.is_read) {
        item.is_read = true;
        item.read_at = readAt;
        updatedCount += 1;
      }
    }
    return {
      updated_count: updatedCount,
      unread_count: state.items.filter(item => !item.is_read).length,
    };
  },

  preference(user: AuthUser): NotificationPreference {
    return { ...stateFor(user).preference };
  },

  replacePreference(
    user: AuthUser,
    preference: NotificationPreference
  ): NotificationPreference {
    stateFor(user).preference = { ...preference };
    return { ...preference };
  },

  reset(): void {
    states.clear();
    nextNotificationId = 1;
  },
};

function stateFor(user: AuthUser): MockNotificationState {
  const existing = states.get(user.id);
  if (existing) return existing;

  const state: MockNotificationState = {
    items: [
      {
        id: nextNotificationId++,
        kind: 'starter.workspace_ready',
        title: 'Your Luas workspace is ready',
        body: 'Replace this development notification with your first downstream workflow.',
        action_url: '/console',
        is_read: true,
        read_at: '2026-07-15T14:10:00Z',
        created_at: '2026-07-15T14:00:00Z',
      },
      {
        id: nextNotificationId++,
        kind: 'starter.notification_ready',
        title: 'Notification starter ready',
        body: 'In-app records, preferences, and durable email delivery are connected.',
        action_url: '/console/settings',
        is_read: false,
        read_at: null,
        created_at: '2026-07-15T14:20:00Z',
      },
    ],
    preference: {
      in_app_enabled: true,
      email_enabled: true,
    },
  };
  states.set(user.id, state);
  return state;
}

function cloneNotification(item: NotificationItem): NotificationItem {
  return { ...item };
}

function pageMeta(total: number, page: number, perPage: number, itemCount: number) {
  const lastPage = Math.max(1, Math.ceil(total / perPage));
  const from = total === 0 ? 0 : (page - 1) * perPage + 1;
  return {
    current_page: page,
    per_page: perPage,
    total,
    last_page: lastPage,
    from,
    to: itemCount === 0 ? 0 : from + itemCount - 1,
  };
}

function pageLinks(
  total: number,
  page: number,
  perPage: number,
  status: NotificationFilter
) {
  const lastPage = Math.max(1, Math.ceil(total / perPage));
  const path = (targetPage: number) =>
    `/api/notifications?page=${targetPage}&per_page=${perPage}&status=${status}`;
  return {
    first: path(1),
    last: path(lastPage),
    prev: page > 1 ? path(page - 1) : null,
    next: page < lastPage ? path(page + 1) : null,
  };
}
