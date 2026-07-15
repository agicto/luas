import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { NextIntlClientProvider, type AbstractIntlMessages } from 'next-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiKeyPanel } from '@/features/api-key/components/api-key-panel';
import { messages } from '@/i18n/modules';

const state = vi.hoisted(() => ({
  apiKeys: {
    data: undefined as unknown,
    isPending: false,
    isError: false,
    refetch: vi.fn(),
  },
  create: {
    isPending: false,
    mutateAsync: vi.fn(),
    reset: vi.fn(),
  },
  revoke: {
    isPending: false,
    mutateAsync: vi.fn(),
  },
}));

vi.mock('@/features/api-key/hooks/use-api-keys', () => ({
  useApiKeys: () => state.apiKeys,
  useCreateApiKey: () => state.create,
  useRevokeApiKey: () => state.revoke,
}));

const apiKey = {
  id: 42,
  user_id: 7,
  name: 'Deployment',
  key_prefix: 'luas_abcdef123456',
  scopes: ['models:read'],
  created_at: '2026-07-15T10:00:00Z',
  updated_at: '2026-07-15T10:00:00Z',
};

describe('API key settings workflow', () => {
  beforeEach(() => {
    state.apiKeys.data = { items: [apiKey], meta: { total: 1 } };
    state.apiKeys.isPending = false;
    state.apiKeys.isError = false;
    state.apiKeys.refetch.mockReset();
    state.create.isPending = false;
    state.create.mutateAsync.mockReset();
    state.create.reset.mockReset();
    state.revoke.isPending = false;
    state.revoke.mutateAsync.mockReset();
  });

  it('renders metadata without fabricating or exposing a plaintext key', () => {
    renderWithMessages(<ApiKeyPanel />);

    expect(screen.getByText('Deployment')).toBeInTheDocument();
    expect(screen.getByText('luas_abcdef123456')).toBeInTheDocument();
    expect(screen.getByText('models:read')).toBeInTheDocument();
    expect(screen.queryByText(/sk_demo/)).not.toBeInTheDocument();
  });

  it('shows a created secret once and resets mutation data immediately', async () => {
    const plaintextKey = 'luas_abcdef123456.0123456789abcdef0123456789abcdef0123456789abcdef';
    state.create.mutateAsync.mockResolvedValue({
      api_key: apiKey,
      plaintext_key: plaintextKey,
    });
    renderWithMessages(<ApiKeyPanel />);

    fireEvent.click(screen.getByRole('button', { name: 'Create key' }));
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Deploy' } });
    fireEvent.change(screen.getByLabelText('Scopes'), {
      target: { value: 'Models:Read\nmodels:invoke' },
    });
    fireEvent.click(screen.getAllByRole('button', { name: 'Create key' }).at(-1)!);

    await waitFor(() => expect(screen.getByDisplayValue(plaintextKey)).toBeInTheDocument());
    expect(state.create.mutateAsync).toHaveBeenCalledWith({
      name: 'Deploy',
      scopes: ['models:read', 'models:invoke'],
    });
    expect(state.create.reset).toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: 'Done' }));
    expect(screen.queryByDisplayValue(plaintextKey)).not.toBeInTheDocument();
  });

  it('requires confirmation before revoking an active key', async () => {
    state.revoke.mutateAsync.mockResolvedValue(undefined);
    renderWithMessages(<ApiKeyPanel />);

    fireEvent.click(screen.getByRole('button', { name: 'Revoke' }));
    expect(state.revoke.mutateAsync).not.toHaveBeenCalled();
    fireEvent.click(screen.getAllByRole('button', { name: 'Revoke' }).at(-1)!);

    await waitFor(() => expect(state.revoke.mutateAsync).toHaveBeenCalledWith(42));
  });

  it('allows an expired key to be explicitly revoked', () => {
    state.apiKeys.data = {
      items: [{ ...apiKey, expires_at: '2020-01-01T00:00:00Z' }],
      meta: { total: 1 },
    };
    renderWithMessages(<ApiKeyPanel />);

    expect(screen.getByRole('button', { name: 'Revoke' })).toBeEnabled();
  });
});

function renderWithMessages(children: React.ReactNode) {
  return render(
    <NextIntlClientProvider locale="en-US" messages={messages as unknown as AbstractIntlMessages}>
      {children}
    </NextIntlClientProvider>
  );
}
