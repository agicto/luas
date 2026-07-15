'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { AlertCircle, Plus } from 'lucide-react';
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
import { useCreateOrganization } from '@/features/organization/hooks/use-organizations';
import { createOrganizationSchema } from '@/features/organization/schemas';
import {
  hasOrganizationFieldError,
  resolveOrganizationErrorKey,
} from '@/features/organization/utils/organization-error';
import { useT } from '@/i18n';

export function CreateOrganizationDialog() {
  const t = useT('organization');
  const common = useT('common');
  const router = useRouter();
  const mutation = useCreateOrganization();
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [clientErrors, setClientErrors] = useState<Record<string, string>>({});

  const resetForm = () => {
    setName('');
    setSlug('');
    setClientErrors({});
    mutation.reset();
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const parsed = createOrganizationSchema.safeParse({
      name,
      ...(slug.trim() ? { slug } : {}),
    });
    if (!parsed.success) {
      const invalidFields = new Set(
        parsed.error.issues.map((issue) => issue.path[0])
      );
      setClientErrors({
        ...(invalidFields.has('name') ? { name: t('nameInvalid') } : {}),
        ...(invalidFields.has('slug') ? { slug: t('slugInvalid') } : {}),
      });
      return;
    }

    setClientErrors({});
    mutation.mutate(parsed.data, {
      onSuccess: (organization) => {
        toast.success(t('createSuccess'));
        setOpen(false);
        resetForm();
        router.push(getOrganizationRoute(organization.id));
      },
    });
  };

  const nameError = clientErrors.name ?? (
    hasOrganizationFieldError(mutation.error, 'name') ? t('nameInvalid') : undefined
  );
  const slugError = clientErrors.slug ?? (
    hasOrganizationFieldError(mutation.error, 'slug') ? t('slugInvalid') : undefined
  );

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen);
        if (!nextOpen) resetForm();
      }}
    >
      <DialogTrigger asChild>
        <Button icon={<Plus className="size-4" />}>{t('create')}</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('create')}</DialogTitle>
          <DialogDescription>{t('createDescription')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} aria-busy={mutation.isPending}>
          <DialogBody className="space-y-4 py-1">
            {mutation.error ? (
              <Alert variant="destructive">
                <AlertCircle aria-hidden="true" className="size-4" />
                <AlertDescription>
                  {t(resolveOrganizationErrorKey(mutation.error))}
                </AlertDescription>
              </Alert>
            ) : null}
            <div className="space-y-2">
              <Label htmlFor="organization-name">{t('name')}</Label>
              <Input
                id="organization-name"
                name="name"
                autoComplete="organization"
                required
                minLength={2}
                maxLength={200}
                value={name}
                errorText={nameError}
                disabled={mutation.isPending}
                onChange={(event) => {
                  setName(event.target.value);
                  setClientErrors((current) => ({ ...current, name: '' }));
                  mutation.reset();
                }}
                placeholder={t('namePlaceholder')}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="organization-slug">{t('slug')}</Label>
              <Input
                id="organization-slug"
                name="slug"
                autoCapitalize="none"
                autoComplete="off"
                spellCheck={false}
                minLength={3}
                maxLength={50}
                value={slug}
                errorText={slugError}
                disabled={mutation.isPending}
                onChange={(event) => {
                  setSlug(event.target.value);
                  setClientErrors((current) => ({ ...current, slug: '' }));
                  mutation.reset();
                }}
                placeholder={t('slugPlaceholder')}
              />
              <p className="text-xs text-muted-foreground">{t('slugHint')}</p>
            </div>
          </DialogBody>
          <DialogFooter className="pt-5">
            <DialogClose asChild>
              <Button type="button" variant="outline" disabled={mutation.isPending}>
                {common('cancel')}
              </Button>
            </DialogClose>
            <Button type="submit" loading={mutation.isPending}>
              {t('create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
