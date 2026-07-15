'use client';

import { useMemo, useState, type FormEvent } from 'react';
import { AlertCircle, Ban, Copy, KeyRound, Plus, RotateCw } from 'lucide-react';
import { useLocale } from 'next-intl';
import { toast } from 'sonner';

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import {
  useApiKeys,
  useCreateApiKey,
  useRevokeApiKey,
} from '@/features/api-key/hooks/use-api-keys';
import { createApiKeySchema } from '@/features/api-key/schemas';
import type { ApiKey, CreateApiKeyResult } from '@/features/api-key/types';
import { useT } from '@/i18n';

export function ApiKeyPanel() {
  const t = useT('settings.api');
  const locale = useLocale();
  const apiKeys = useApiKeys();
  const createApiKey = useCreateApiKey();
  const revokeApiKey = useRevokeApiKey();
  const [createOpen, setCreateOpen] = useState(false);
  const [created, setCreated] = useState<CreateApiKeyResult | null>(null);
  const [name, setName] = useState('');
  const [scopes, setScopes] = useState('');
  const [expiresAt, setExpiresAt] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<ApiKey | null>(null);
  const dateFormatter = useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }),
    [locale]
  );

  const closeCreateDialog = () => {
    setCreateOpen(false);
    setCreated(null);
    setName('');
    setScopes('');
    setExpiresAt('');
    setFormError(null);
    createApiKey.reset();
  };

  const submitCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setFormError(null);
    const expiry = expiresAt ? new Date(expiresAt) : null;
    if (expiry && (Number.isNaN(expiry.getTime()) || expiry <= new Date())) {
      setFormError(t('invalidForm'));
      return;
    }
    const candidate = {
      name,
      scopes: scopes
        .split(/\r?\n/)
        .map(scope => scope.trim())
        .filter(Boolean),
      ...(expiry ? { expires_at: expiry.toISOString() } : {}),
    };
    const parsed = createApiKeySchema.safeParse(candidate);
    if (!parsed.success) {
      setFormError(t('invalidForm'));
      return;
    }

    try {
      const result = await createApiKey.mutateAsync(parsed.data);
      setCreated(result);
      createApiKey.reset();
      toast.success(t('createdSuccess'));
    } catch {
      createApiKey.reset();
      setFormError(t('createError'));
    }
  };

  const copySecret = async () => {
    if (!created) return;
    try {
      await navigator.clipboard.writeText(created.plaintext_key);
      toast.success(t('copied'));
    } catch {
      toast.error(t('copyError'));
    }
  };

  const confirmRevoke = async () => {
    if (!revokeTarget) return;
    try {
      await revokeApiKey.mutateAsync(revokeTarget.id);
      toast.success(t('revokedSuccess'));
      setRevokeTarget(null);
    } catch {
      toast.error(t('revokeError'));
    }
  };

  return (
    <>
      <Card>
        <CardHeader className="gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="space-y-1.5">
            <CardTitle>{t('title')}</CardTitle>
            <CardDescription>{t('description')}</CardDescription>
          </div>
          <Button size="sm" icon={<Plus className="size-4" />} onClick={() => setCreateOpen(true)}>
            {t('create')}
          </Button>
        </CardHeader>
        <CardContent>
          {apiKeys.isPending ? (
            <div className="flex min-h-40 items-center justify-center text-sm text-muted-foreground">
              {t('loading')}
            </div>
          ) : apiKeys.isError ? (
            <Alert variant="destructive">
              <AlertCircle className="size-4" />
              <AlertTitle>{t('loadErrorTitle')}</AlertTitle>
              <AlertDescription className="flex items-center justify-between gap-4">
                <span>{t('loadError')}</span>
                <Button
                  size="xs"
                  variant="outline"
                  icon={<RotateCw className="size-3" />}
                  onClick={() => apiKeys.refetch()}
                >
                  {t('retry')}
                </Button>
              </AlertDescription>
            </Alert>
          ) : apiKeys.data.items.length === 0 ? (
            <div className="flex min-h-40 flex-col items-center justify-center gap-3 border border-dashed p-6 text-center">
              <KeyRound className="size-7 text-muted-foreground" />
              <div>
                <p className="text-sm font-medium">{t('emptyTitle')}</p>
                <p className="mt-1 text-sm text-muted-foreground">{t('emptyDescription')}</p>
              </div>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('columns.name')}</TableHead>
                  <TableHead>{t('columns.scopes')}</TableHead>
                  <TableHead>{t('columns.status')}</TableHead>
                  <TableHead>{t('columns.lastUsed')}</TableHead>
                  <TableHead className="text-right">{t('columns.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {apiKeys.data.items.map(apiKey => {
                  const status = apiKeyStatus(apiKey);
                  return (
                    <TableRow key={apiKey.id}>
                      <TableCell>
                        <div className="font-medium">{apiKey.name}</div>
                        <code className="text-xs text-muted-foreground">{apiKey.key_prefix}</code>
                      </TableCell>
                      <TableCell>
                        <div className="flex max-w-72 flex-wrap gap-1">
                          {apiKey.scopes.length ? (
                            apiKey.scopes.map(scope => (
                              <Badge key={scope} variant="outline">
                                {scope}
                              </Badge>
                            ))
                          ) : (
                            <span className="text-muted-foreground">{t('noScopes')}</span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge variant={status === 'revoked' ? 'destructive' : 'outline'}>
                          {t(`status.${status}`)}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {apiKey.last_used_at
                          ? dateFormatter.format(new Date(apiKey.last_used_at))
                          : t('neverUsed')}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          isIcon
                          size="sm"
                          variant="ghost"
                          aria-label={t('revoke')}
                          title={t('revoke')}
                          disabled={status === 'revoked'}
                          onClick={() => setRevokeTarget(apiKey)}
                        >
                          <Ban className="size-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={createOpen}
        onOpenChange={open => (open ? setCreateOpen(true) : closeCreateDialog())}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{created ? t('secretTitle') : t('createTitle')}</DialogTitle>
            <DialogDescription>
              {created ? t('secretDescription') : t('createDescription')}
            </DialogDescription>
          </DialogHeader>
          {created ? (
            <>
              <DialogBody className="space-y-4">
                <Alert>
                  <KeyRound className="size-4" />
                  <AlertTitle>{t('secretWarningTitle')}</AlertTitle>
                  <AlertDescription>{t('secretWarning')}</AlertDescription>
                </Alert>
                <div className="space-y-2">
                  <Label htmlFor="created-api-key">{t('plaintextKey')}</Label>
                  <div className="flex gap-2">
                    <Input id="created-api-key" value={created.plaintext_key} readOnly />
                    <Button
                      type="button"
                      isIcon
                      variant="outline"
                      aria-label={t('copy')}
                      title={t('copy')}
                      onClick={copySecret}
                    >
                      <Copy className="size-4" />
                    </Button>
                  </div>
                </div>
              </DialogBody>
              <DialogFooter>
                <Button onClick={closeCreateDialog}>{t('done')}</Button>
              </DialogFooter>
            </>
          ) : (
            <form onSubmit={submitCreate}>
              <DialogBody className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="api-key-name">{t('name')}</Label>
                  <Input
                    id="api-key-name"
                    value={name}
                    maxLength={100}
                    autoComplete="off"
                    placeholder={t('namePlaceholder')}
                    onChange={event => setName(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="api-key-scopes">{t('scopes')}</Label>
                  <Textarea
                    id="api-key-scopes"
                    value={scopes}
                    rows={4}
                    spellCheck={false}
                    placeholder={t('scopesPlaceholder')}
                    onChange={event => setScopes(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="api-key-expiry">{t('expiresAt')}</Label>
                  <Input
                    id="api-key-expiry"
                    type="datetime-local"
                    value={expiresAt}
                    onChange={event => setExpiresAt(event.target.value)}
                  />
                </div>
                {formError ? (
                  <p role="alert" className="text-sm text-error">
                    {formError}
                  </p>
                ) : null}
              </DialogBody>
              <DialogFooter className="mt-4">
                <Button type="button" variant="outline" onClick={closeCreateDialog}>
                  {t('cancel')}
                </Button>
                <Button type="submit" loading={createApiKey.isPending}>
                  {t('create')}
                </Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={revokeTarget !== null} onOpenChange={open => !open && setRevokeTarget(null)}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>{t('revokeTitle')}</DialogTitle>
            <DialogDescription>{t('revokeDescription')}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRevokeTarget(null)}>
              {t('cancel')}
            </Button>
            <Button variant="destructive" loading={revokeApiKey.isPending} onClick={confirmRevoke}>
              {t('revoke')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function apiKeyStatus(apiKey: ApiKey): 'active' | 'expired' | 'revoked' {
  if (apiKey.revoked_at) return 'revoked';
  if (apiKey.expires_at && new Date(apiKey.expires_at) <= new Date()) return 'expired';
  return 'active';
}
