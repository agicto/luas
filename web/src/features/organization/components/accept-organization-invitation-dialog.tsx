'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { KeyRound } from 'lucide-react';
import { toast } from 'sonner';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogBody,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { getOrganizationRoute } from '@/constants/routes';
import { useAcceptOrganizationInvitation } from '@/features/organization/hooks/use-organizations';
import { acceptOrganizationInvitationSchema } from '@/features/organization/schemas';
import {
  hasOrganizationFieldError,
  resolveOrganizationErrorKey,
} from '@/features/organization/utils/organization-error';
import { useT } from '@/i18n';

export function AcceptOrganizationInvitationDialog() {
  const t = useT('organization');
  const common = useT('common');
  const router = useRouter();
  const mutation = useAcceptOrganizationInvitation();
  const [open, setOpen] = useState(false);
  const [token, setToken] = useState('');
  const [clientError, setClientError] = useState<string>();

  const reset = () => {
    setToken('');
    setClientError(undefined);
    mutation.reset();
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const parsed = acceptOrganizationInvitationSchema.safeParse({ token });
    if (!parsed.success) {
      setClientError(t('invitationTokenInvalid'));
      return;
    }
    setClientError(undefined);
    mutation.mutate(parsed.data, {
      onSuccess: (organization) => {
        toast.success(t('acceptInvitationSuccess'));
        setOpen(false);
        reset();
        router.push(getOrganizationRoute(organization.id));
      },
    });
  };

  const tokenError =
    clientError ??
    (hasOrganizationFieldError(mutation.error, 'token')
      ? t('invitationTokenInvalid')
      : undefined);

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (!nextOpen) reset();
      }}
    >
      <DialogTrigger asChild>
        <Button variant="outline" icon={<KeyRound className="size-4" />}>
          {t('acceptInvitation')}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('acceptInvitation')}</DialogTitle>
          <DialogDescription>{t('acceptInvitationDescription')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} aria-busy={mutation.isPending}>
          <DialogBody className="space-y-4 py-1">
            {mutation.error ? (
              <Alert variant="destructive">
                <AlertDescription>
                  {t(resolveOrganizationErrorKey(mutation.error))}
                </AlertDescription>
              </Alert>
            ) : null}
            <div className="space-y-2">
              <Label htmlFor="organization-invitation-token">
                {t('invitationToken')}
              </Label>
              <Input
                id="organization-invitation-token"
                name="token"
                type="password"
                autoComplete="off"
                autoCapitalize="none"
                spellCheck={false}
                required
                maxLength={256}
                value={token}
                errorText={tokenError}
                disabled={mutation.isPending}
                placeholder={t('invitationTokenPlaceholder')}
                onChange={(event) => {
                  setToken(event.target.value);
                  setClientError(undefined);
                  mutation.reset();
                }}
              />
            </div>
          </DialogBody>
          <DialogFooter className="pt-5">
            <DialogClose asChild>
              <Button type="button" variant="outline" disabled={mutation.isPending}>
                {common('cancel')}
              </Button>
            </DialogClose>
            <Button type="submit" loading={mutation.isPending}>
              {t('acceptInvitation')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
