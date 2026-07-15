import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { NextIntlClientProvider, type AbstractIntlMessages } from 'next-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { WebhookManagement } from '@/features/webhook/components/webhook-management';
import { messages } from '@/i18n/modules';

const state = vi.hoisted(() => ({
  endpoints: queryState(),
  deliveries: queryState(),
  attempts: queryState(),
  eventTypes: {
    data: ['webhook.test'],
    error: null as unknown,
    isError: false,
  },
  create: asyncMutationState(),
  update: asyncMutationState(),
  remove: asyncMutationState(),
  rotate: asyncMutationState(),
  status: mutationState(),
  test: mutationState(),
}));

vi.mock('@/features/webhook/hooks/use-webhooks', () => ({
  useWebhookEndpoints: () => state.endpoints,
  useWebhookDeliveries: () => state.deliveries,
  useWebhookAttempts: () => state.attempts,
  useWebhookEventTypes: () => state.eventTypes,
  useCreateWebhookEndpoint: () => state.create,
  useUpdateWebhookEndpoint: () => state.update,
  useDeleteWebhookEndpoint: () => state.remove,
  useRotateWebhookEndpointSecret: () => state.rotate,
  useReplaceWebhookEndpointStatus: () => state.status,
  useTestWebhookEndpoint: () => state.test,
}));

describe('webhook management workflow', () => {
  beforeEach(() => {
    state.endpoints.data = page([endpoint]);
    state.endpoints.error = null;
    state.endpoints.isPending = false;
    state.endpoints.isFetching = false;
    state.endpoints.refetch.mockReset();
    state.deliveries.data = page([delivery]);
    state.deliveries.error = null;
    state.deliveries.isPending = false;
    state.deliveries.isFetching = false;
    state.deliveries.refetch.mockReset();
    state.attempts.data = page([]);
    state.attempts.error = null;
    state.attempts.isPending = false;
    state.attempts.isFetching = false;
    state.attempts.refetch.mockReset();
    state.eventTypes.data = ['webhook.test'];
    state.eventTypes.error = null;
    state.eventTypes.isError = false;
    for (const mutation of [state.create, state.update, state.remove, state.rotate]) {
      mutation.error = null;
      mutation.isPending = false;
      mutation.mutateAsync.mockReset();
      mutation.reset.mockReset();
    }
    for (const mutation of [state.status, state.test]) {
      mutation.isPending = false;
      mutation.mutate.mockReset();
    }
  });

  it('renders endpoint and minimized delivery metadata without exposing secret material', () => {
    renderWithMessages(<WebhookManagement organizationId={7} />);

    expect(screen.getByText('Order processor')).toBeInTheDocument();
    expect(screen.getByText('https://hooks.example.com/luas')).toBeInTheDocument();
    expect(screen.getAllByText('webhook.test')).not.toHaveLength(0);
    expect(screen.getByText('WEBHOOK.MOCK_NOT_DELIVERED')).toBeInTheDocument();
    expect(screen.queryByText(/whsec_/u)).not.toBeInTheDocument();
    expect(screen.queryByText(/ciphertext/i)).not.toBeInTheDocument();
  });

  it('shows a created signing secret once and clears mutation data immediately', async () => {
    const signingSecret = `whsec_${'A'.repeat(43)}=`;
    state.create.mutateAsync.mockResolvedValue({
      endpoint,
      signing_secret: signingSecret,
      previous_secret_expiry: null,
    });
    renderWithMessages(<WebhookManagement organizationId={7} />);

    fireEvent.click(screen.getByRole('button', { name: 'Create endpoint' }));
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Billing consumer' } });
    fireEvent.change(screen.getByLabelText('Target URL'), {
      target: { value: 'https://billing.example.com/hooks' },
    });
    fireEvent.click(screen.getAllByRole('button', { name: 'Create endpoint' }).at(-1)!);

    await waitFor(() => expect(screen.getByDisplayValue(signingSecret)).toBeInTheDocument());
    expect(state.create.mutateAsync).toHaveBeenCalledWith({
      name: 'Billing consumer',
      url: 'https://billing.example.com/hooks',
      event_types: ['webhook.test'],
    });
    expect(state.create.reset).toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: 'Done' }));
    expect(screen.queryByDisplayValue(signingSecret)).not.toBeInTheDocument();
  });

  it('queues only a fixed endpoint test with a fresh canonical idempotency key', () => {
    renderWithMessages(<WebhookManagement organizationId={7} />);

    fireEvent.click(screen.getByRole('button', { name: 'Send test event' }));

    expect(state.test.mutate).toHaveBeenCalledWith(
      {
        endpoint,
        idempotencyKey: expect.stringMatching(/^ui-[0-9a-f-]{36}$/u),
      },
      expect.objectContaining({ onSuccess: expect.any(Function), onError: expect.any(Function) })
    );
  });
});

const endpoint = {
  id: 11,
  organization_id: 7,
  name: 'Order processor',
  url: 'https://hooks.example.com/luas',
  event_types: ['webhook.test'] as const,
  status: 'active' as const,
  disabled_reason: '',
  consecutive_failures: 0,
  version: 3,
  secret_hint: 'abcd1234',
  secret_version: 1,
  previous_secret_expiry: null,
  created_at: '2026-07-15T10:00:00Z',
  updated_at: '2026-07-15T10:00:00Z',
};

const delivery = {
  id: 81,
  endpoint_id: 11,
  message_id: 'msg_01JTEST',
  event_type: 'webhook.test' as const,
  status: 'canceled' as const,
  attempt_count: 0,
  replay_count: 0,
  http_status: null,
  failure_code: 'WEBHOOK.MOCK_NOT_DELIVERED',
  response_truncated: false,
  available_at: '2026-07-15T10:00:00Z',
  delivered_at: null,
  created_at: '2026-07-15T10:00:00Z',
  updated_at: '2026-07-15T10:00:00Z',
};

function page<T>(items: T[]) {
  return {
    items,
    meta: {
      current_page: 1,
      per_page: 25,
      total: items.length,
      last_page: 1,
      from: items.length ? 1 : 0,
      to: items.length,
    },
    links: { first: '', last: '', prev: null, next: null },
  };
}

function queryState() {
  return {
    data: undefined as unknown,
    error: null as unknown,
    isPending: false,
    isFetching: false,
    refetch: vi.fn(),
  };
}

function asyncMutationState() {
  return {
    error: null as unknown,
    isPending: false,
    mutateAsync: vi.fn(),
    reset: vi.fn(),
  };
}

function mutationState() {
  return {
    isPending: false,
    mutate: vi.fn(),
  };
}

function renderWithMessages(children: React.ReactNode) {
  return render(
    <NextIntlClientProvider locale="en-US" messages={messages as unknown as AbstractIntlMessages}>
      {children}
    </NextIntlClientProvider>
  );
}
