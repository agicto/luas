'use client';

import { useState } from 'react';
import { MailPlus } from 'lucide-react';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useCreateOrganizationInvitation } from '@/features/organization/hooks/use-organizations';
import { createOrganizationInvitationSchema } from '@/features/organization/schemas';
import type { OrganizationInvitationRole } from '@/features/organization/types';
import {
  hasOrganizationFieldError,
  resolveOrganizationErrorKey,
} from '@/features/organization/utils/organization-error';
import { useT } from '@/i18n';

export function CreateOrganizationInvitationDialog({
  organizationId,
}: {
  organizationId: number;
}) {
  const t = useT('organization');
  const common = useT('common');
  const mutation = useCreateOrganizationInvitation(organizationId);
  const [open, setOpen] = useState(false);
  const [email, setEmail] = useState('');
  const [role, setRole] = useState<OrganizationInvitationRole>('member');
  const [clientErrors, setClientErrors] = useState<Record<string, string>>({});

  const reset = () => {
    setEmail('');
    setRole('member');
    setClientErrors({});
    mutation.reset();
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const parsed = createOrganizationInvitationSchema.safeParse({ email, role });
    if (!parsed.success) {
      const invalidFields = new Set(
        parsed.error.issues.map((issue) => issue.path[0])
      );
      setClientErrors({
        ...(invalidFields.has('email') ? { email: t('emailInvalid') } : {}),
      });
      return;
    }

    setClientErrors({});
    mutation.mutate(parsed.data, {
      onSuccess: ({ email_send_status: status }) => {
        toast.success(t('invitationCreated'), {
          description: t(`emailSendStatus.${status}`),
        });
        setOpen(false);
        reset();
      },
    });
  };

  const emailError =
    clientErrors.email ??
    (hasOrganizationFieldError(mutation.error, 'email')
      ? t('emailInvalid')
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
        <Button icon={<MailPlus className="size-4" />}>{t('invite')}</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('invite')}</DialogTitle>
          <DialogDescription>{t('inviteDescription')}</DialogDescription>
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
              <Label htmlFor="organization-invitation-email">{t('email')}</Label>
              <Input
                id="organization-invitation-email"
                name="email"
                type="email"
                autoComplete="email"
                autoCapitalize="none"
                spellCheck={false}
                required
                maxLength={100}
                value={email}
                errorText={emailError}
                disabled={mutation.isPending}
                placeholder={t('emailPlaceholder')}
                onChange={(event) => {
                  setEmail(event.target.value);
                  setClientErrors({});
                  mutation.reset();
                }}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="organization-invitation-role">
                {t('invitationRole')}
              </Label>
              <Select
                value={role}
                disabled={mutation.isPending}
                onValueChange={(value) => {
                  setRole(value as OrganizationInvitationRole);
                  mutation.reset();
                }}
              >
                <SelectTrigger id="organization-invitation-role" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="member">{t('roles.member')}</SelectItem>
                  <SelectItem value="admin">{t('roles.admin')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </DialogBody>
          <DialogFooter className="pt-5">
            <DialogClose asChild>
              <Button type="button" variant="outline" disabled={mutation.isPending}>
                {common('cancel')}
              </Button>
            </DialogClose>
            <Button type="submit" loading={mutation.isPending}>
              {t('sendInvitation')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
