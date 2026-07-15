import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { NextIntlClientProvider, type AbstractIntlMessages } from 'next-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { AssetPanel } from '@/features/asset/components/asset-panel';
import { messages } from '@/i18n/modules';

const toast = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
const state = vi.hoisted(() => ({
  assets: {
    data: undefined as unknown,
    isPending: false,
    isError: false,
    refetch: vi.fn(),
  },
  upload: { isPending: false, mutateAsync: vi.fn() },
  download: { isPending: false, mutateAsync: vi.fn() },
  remove: { isPending: false, mutateAsync: vi.fn() },
}));

vi.mock('sonner', () => ({ toast }));
vi.mock('@/features/asset/hooks/use-assets', () => ({
  useAssets: () => state.assets,
  useUploadAsset: () => state.upload,
  useDownloadAsset: () => state.download,
  useDeleteAsset: () => state.remove,
}));

const readyAsset = {
  id: '019bf6d8-17c5-7a98-a084-6d45793f5f0c',
  original_name: 'report.pdf',
  media_type: 'application/pdf',
  size_bytes: 1_024,
  status: 'ready' as const,
  created_at: '2026-07-15T20:00:00Z',
  ready_at: '2026-07-15T20:00:04Z',
};

const pendingAsset = {
  ...readyAsset,
  id: '019bf6d8-17c5-7a98-a084-6d45793f5f0d',
  original_name: 'pending.txt',
  media_type: 'text/plain',
  status: 'pending' as const,
  ready_at: null,
};

describe('asset console workflow', () => {
  beforeEach(() => {
    state.assets.data = { items: [readyAsset, pendingAsset], meta: { total: 2 } };
    state.assets.isPending = false;
    state.assets.isError = false;
    state.assets.refetch.mockReset();
    state.upload.isPending = false;
    state.upload.mutateAsync.mockReset();
    state.download.isPending = false;
    state.download.mutateAsync.mockReset();
    state.download.mutateAsync.mockResolvedValue(undefined);
    state.remove.isPending = false;
    state.remove.mutateAsync.mockReset();
    state.remove.mutateAsync.mockResolvedValue(undefined);
    toast.success.mockReset();
    toast.error.mockReset();
  });

  it('renders private metadata and disables download until an asset is ready', () => {
    renderWithMessages(<AssetPanel />);

    expect(screen.getByText('report.pdf')).toBeInTheDocument();
    expect(screen.getByText('pending.txt')).toBeInTheDocument();
    const downloads = screen.getAllByRole('button', { name: 'Download' });
    expect(downloads[0]).toBeEnabled();
    expect(downloads[1]).toBeDisabled();
    expect(screen.queryByText(/object_key|checksum|asset-uploads/)).not.toBeInTheDocument();
  });

  it('rejects unsupported files before creating an upload intent', () => {
    const view = renderWithMessages(<AssetPanel />);
    const input = view.container.querySelector('input[type="file"]');
    expect(input).not.toBeNull();

    fireEvent.change(input!, {
      target: { files: [new File(['<svg/>'], 'unsafe.svg', { type: 'image/svg+xml' })] },
    });

    expect(state.upload.mutateAsync).not.toHaveBeenCalled();
    expect(toast.error).toHaveBeenCalledWith('The file type or size is not allowed.');
  });

  it('downloads ready assets and requires confirmation before deletion', async () => {
    renderWithMessages(<AssetPanel />);

    fireEvent.click(screen.getAllByRole('button', { name: 'Download' })[0]);
    await waitFor(() => expect(state.download.mutateAsync).toHaveBeenCalledWith(readyAsset));

    fireEvent.click(screen.getAllByRole('button', { name: 'Delete' })[0]);
    expect(state.remove.mutateAsync).not.toHaveBeenCalled();
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Delete asset' }));

    await waitFor(() => expect(state.remove.mutateAsync).toHaveBeenCalledWith(readyAsset.id));
    expect(toast.success).toHaveBeenCalledWith('Asset deleted');
  });
});

function renderWithMessages(children: React.ReactNode) {
  return render(
    <NextIntlClientProvider locale="en-US" messages={messages as unknown as AbstractIntlMessages}>
      {children}
    </NextIntlClientProvider>
  );
}
