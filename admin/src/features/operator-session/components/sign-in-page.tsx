import { useNavigate } from '@tanstack/react-router';
import { LogIn, ShieldCheck } from 'lucide-react';
import { type FormEvent, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useSignIn } from '@/features/operator-session/hooks/use-operator-session';
import { signInErrorKey } from '@/features/operator-session/sign-in-errors';

export function SignInPage({ redirectTo }: { redirectTo: string }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const signIn = useSignIn();
  const [identifier, setIdentifier] = useState('');
  const [password, setPassword] = useState('');

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    signIn.mutate(
      { identifier: identifier.trim(), password },
      {
        onSuccess: () => {
          setPassword('');
          void navigate({ to: redirectTo, replace: true });
        },
      },
    );
  };

  return (
    <main className="flex min-h-svh items-center justify-center bg-background px-4 py-10">
      <section className="panel w-full max-w-sm" aria-labelledby="sign-in-title">
        <div className="panel-header">
          <div className="flex items-start gap-3">
            <span className="icon-surface" aria-hidden="true">
              <ShieldCheck className="size-4" />
            </span>
            <div>
              <h1 id="sign-in-title" className="panel-title">
                {t('auth.title')}
              </h1>
              <p className="panel-description">{t('auth.description')}</p>
            </div>
          </div>
        </div>
        <form className="panel-body flex flex-col gap-4" onSubmit={submit} noValidate>
          <label className="flex flex-col gap-1.5">
            <span className="field-label">{t('auth.identifier')}</span>
            <Input
              name="identifier"
              autoComplete="username"
              required
              maxLength={100}
              value={identifier}
              onChange={(event) => setIdentifier(event.target.value)}
            />
          </label>
          <label className="flex flex-col gap-1.5">
            <span className="field-label">{t('auth.password')}</span>
            <Input
              name="password"
              type="password"
              autoComplete="current-password"
              required
              maxLength={128}
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </label>
          {signIn.isError ? (
            <p role="alert" className="text-sm text-destructive">
              {t(signInErrorKey(signIn.error))}
            </p>
          ) : null}
          <Button
            type="submit"
            disabled={signIn.isPending || identifier.trim() === '' || password === ''}
          >
            <LogIn aria-hidden="true" />
            {signIn.isPending ? t('auth.submitting') : t('auth.submit')}
          </Button>
          <p className="text-xs text-muted-foreground">{t('auth.operatorNote')}</p>
        </form>
      </section>
    </main>
  );
}
