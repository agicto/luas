'use client';

import { useMemo, useRef, useState } from 'react';
import { useLocale } from 'next-intl';
import { Download, FileText, FolderOpen, Loader2, RefreshCw, Trash2, Upload } from 'lucide-react';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
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
  useAssets,
  useDeleteAsset,
  useDownloadAsset,
  useUploadAsset,
} from '@/features/asset/hooks/use-assets';
import { ASSET_DEFAULT_UPLOAD_BYTES, isAllowedAssetFile } from '@/features/asset/schemas';
import type { AssetFilter, AssetItem, AssetStatus } from '@/features/asset/types';
import { useT } from '@/i18n';

const acceptedAssetTypes = '.jpg,.jpeg,.png,.webp,.pdf,.txt,.csv';

export function AssetPanel() {
  const t = useT('asset');
  const locale = useLocale();
  const inputRef = useRef<HTMLInputElement>(null);
  const [filter, setFilter] = useState<AssetFilter>('all');
  const [deleteTarget, setDeleteTarget] = useState<AssetItem | null>(null);
  const assets = useAssets(filter);
  const uploadAsset = useUploadAsset();
  const downloadAsset = useDownloadAsset();
  const deleteAsset = useDeleteAsset();
  const dateFormatter = useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }),
    [locale]
  );

  async function uploadFile(file: File): Promise<void> {
    if (file.size > ASSET_DEFAULT_UPLOAD_BYTES || !isAllowedAssetFile(file)) {
      toast.error(t('invalidFile'));
      return;
    }
    try {
      await uploadAsset.mutateAsync({ file, idempotencyKey: crypto.randomUUID() });
      toast.success(t('uploaded'));
    } catch {
      toast.error(t('uploadError'));
    }
  }

  async function download(item: AssetItem): Promise<void> {
    try {
      await downloadAsset.mutateAsync(item);
    } catch {
      toast.error(t('downloadError'));
    }
  }

  async function confirmDelete(): Promise<void> {
    if (!deleteTarget) return;
    try {
      await deleteAsset.mutateAsync(deleteTarget.id);
      toast.success(t('deleted'));
      setDeleteTarget(null);
    } catch {
      toast.error(t('deleteError'));
    }
  }

  return (
    <div className="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
      <div className="flex flex-col gap-4 border-b pb-5 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <h1 className="text-2xl font-semibold">{t('title')}</h1>
          <p className="mt-1 text-sm text-text-muted">
            {t('count', { count: assets.data?.meta.total ?? 0 })}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Select value={filter} onValueChange={value => setFilter(value as AssetFilter)}>
            <SelectTrigger className="min-w-32" aria-label={t('filterLabel')}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(['all', 'pending', 'ready', 'rejected'] as const).map(value => (
                <SelectItem key={value} value={value}>
                  {t(`filter.${value}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <input
            ref={inputRef}
            type="file"
            className="sr-only"
            accept={acceptedAssetTypes}
            onChange={event => {
              const file = event.target.files?.[0];
              event.target.value = '';
              if (file) void uploadFile(file);
            }}
          />
          <Button
            type="button"
            icon={<Upload className="size-4" />}
            loading={uploadAsset.isPending}
            onClick={() => inputRef.current?.click()}
          >
            {t('upload')}
          </Button>
        </div>
      </div>

      <div className="mt-5 overflow-hidden border">
        {assets.isPending ? (
          <PanelState icon={Loader2} text={t('loading')} spin />
        ) : assets.isError ? (
          <div className="flex min-h-56 flex-col items-center justify-center gap-3 p-6 text-center">
            <p className="text-sm text-text-muted">{t('loadError')}</p>
            <Button
              type="button"
              size="sm"
              variant="outline"
              icon={<RefreshCw className="size-4" />}
              onClick={() => assets.refetch()}
            >
              {t('retry')}
            </Button>
          </div>
        ) : assets.data.items.length === 0 ? (
          <PanelState icon={FolderOpen} text={t('empty')} />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('columns.name')}</TableHead>
                <TableHead>{t('columns.type')}</TableHead>
                <TableHead>{t('columns.size')}</TableHead>
                <TableHead>{t('columns.status')}</TableHead>
                <TableHead>{t('columns.created')}</TableHead>
                <TableHead className="text-right">{t('columns.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {assets.data.items.map(item => (
                <TableRow key={item.id}>
                  <TableCell>
                    <div className="flex max-w-80 items-center gap-2">
                      <FileText className="size-4 shrink-0 text-text-muted" aria-hidden="true" />
                      <span className="truncate font-medium" title={item.original_name}>
                        {item.original_name}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="text-text-muted">{item.media_type}</TableCell>
                  <TableCell className="text-text-muted">{formatBytes(item.size_bytes)}</TableCell>
                  <TableCell>
                    <AssetStatusBadge status={item.status} />
                  </TableCell>
                  <TableCell className="text-text-muted">
                    {dateFormatter.format(new Date(item.created_at))}
                  </TableCell>
                  <TableCell>
                    <div className="flex justify-end gap-1">
                      <IconAction
                        label={t('download')}
                        disabled={item.status !== 'ready' || downloadAsset.isPending}
                        onClick={() => void download(item)}
                      >
                        <Download className="size-4" />
                      </IconAction>
                      <IconAction
                        label={t('delete')}
                        destructive
                        disabled={deleteAsset.isPending}
                        onClick={() => setDeleteTarget(item)}
                      >
                        <Trash2 className="size-4" />
                      </IconAction>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>

      <p className="sr-only" aria-live="polite">
        {uploadAsset.isPending ? t('uploading') : ''}
      </p>

      <Dialog open={deleteTarget !== null} onOpenChange={open => !open && setDeleteTarget(null)}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>{t('deleteTitle')}</DialogTitle>
            <DialogDescription>{t('deleteDescription')}</DialogDescription>
          </DialogHeader>
          <DialogBody>
            <p className="break-all text-sm font-medium">{deleteTarget?.original_name}</p>
          </DialogBody>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setDeleteTarget(null)}>
              {t('cancel')}
            </Button>
            <Button
              type="button"
              variant="destructive"
              loading={deleteAsset.isPending}
              onClick={() => void confirmDelete()}
            >
              {t('confirmDelete')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function AssetStatusBadge({ status }: { status: AssetStatus }) {
  const t = useT('asset.status');
  const classes: Record<AssetStatus, string> = {
    pending: 'border-warning/40 text-warning',
    ready: 'border-success/40 text-success',
    rejected: 'border-error/40 text-error',
  };
  return (
    <Badge variant="outline" className={classes[status]}>
      {t(status)}
    </Badge>
  );
}

function IconAction({
  label,
  destructive = false,
  children,
  ...props
}: React.ComponentProps<typeof Button> & { label: string; destructive?: boolean }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          isIcon
          size="sm"
          variant="ghost"
          aria-label={label}
          className={destructive ? 'text-error hover:bg-destructive/10' : undefined}
          {...props}
        >
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

function PanelState({
  icon: Icon,
  text,
  spin = false,
}: {
  icon: typeof FolderOpen;
  text: string;
  spin?: boolean;
}) {
  return (
    <div className="flex min-h-56 flex-col items-center justify-center gap-3 p-6 text-center text-text-muted">
      <Icon className={`size-7${spin ? ' animate-spin' : ''}`} aria-hidden="true" />
      <p className="text-sm">{text}</p>
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes < 1_024) return `${bytes} B`;
  if (bytes < 1_024 * 1_024) return `${(bytes / 1_024).toFixed(1)} KB`;
  return `${(bytes / (1_024 * 1_024)).toFixed(1)} MB`;
}
