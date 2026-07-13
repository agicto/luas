'use client';

import { useState } from 'react';
import Link from 'next/link';
import { AlertCircle } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useLogin } from '@/features/auth/hooks/use-auth';
import { hasAuthFieldError, resolveAuthErrorKey } from '@/features/auth/utils/auth-error';
import { useT } from '@/i18n';

interface LoginFormProps {
  demoCredentials?: {
    email: string;
    password: string;
  };
}

/** Login entry point with localized, contract-aware mutation feedback. */
export function LoginForm({ demoCredentials }: LoginFormProps) {
  const t = useT();
  const { mutate: login, isPending, error, reset } = useLogin();
  const [email, setEmail] = useState(demoCredentials?.email ?? '');
  const [password, setPassword] = useState(demoCredentials?.password ?? '');
  const emailError = hasAuthFieldError(error, 'email') ? t.auth('emailInvalid') : undefined;
  const passwordError = hasAuthFieldError(error, 'password')
    ? t.auth('passwordInvalid')
    : undefined;

  const clearMutationError = () => {
    if (error) {
      reset();
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    login({ email, password });
  };

  return (
    <Card className="border-none shadow-none bg-transparent">
      <CardHeader className="space-y-1 px-0">
        <CardTitle className="text-2xl font-bold tracking-tight">{t.auth('signIn')}</CardTitle>
        <CardDescription>{t.auth('signInToContinue')}</CardDescription>
      </CardHeader>
      <CardContent className="px-0 pt-4">
        <form onSubmit={handleSubmit} className="space-y-4" aria-busy={isPending}>
          {error && (
            <Alert variant="destructive">
              <AlertCircle aria-hidden="true" className="size-4" />
              <AlertDescription>{t(resolveAuthErrorKey(error, 'login'))}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="email">{t.auth('email')}</Label>
            <Input
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              placeholder={t.auth('enterEmail')}
              required
              disabled={isPending}
              className="bg-bg-canvas"
              value={email}
              errorText={emailError}
              onChange={event => {
                setEmail(event.target.value);
                clearMutationError();
              }}
            />
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label htmlFor="password">{t.auth('password')}</Label>
              {demoCredentials && (
                <span className="text-sm text-text-muted">
                  {t.auth('demoCredentialsHint')}
                </span>
              )}
            </div>
            <Input
              id="password"
              name="password"
              type="password"
              autoComplete="current-password"
              required
              disabled={isPending}
              className="bg-bg-canvas"
              value={password}
              errorText={passwordError}
              onChange={event => {
                setPassword(event.target.value);
                clearMutationError();
              }}
            />
          </div>
          <Button type="submit" className="w-full font-semibold" loading={isPending}>
            {t.auth('login')}
          </Button>
        </form>

        {demoCredentials && (
          <div className="mt-4 rounded-lg border border-border/60 bg-bg-surface/80 p-3 text-sm text-text-subtle">
            <p className="font-medium text-text-main">{t.auth('demoCredentials')}</p>
            <p>
              {demoCredentials.email} / {demoCredentials.password}
            </p>
          </div>
        )}

        <div className="relative my-6">
          <div className="absolute inset-0 flex items-center">
            <span className="w-full border-t" />
          </div>
          <div className="relative flex justify-center text-xs uppercase">
            <span className="bg-background px-2 text-muted-foreground">
              {t.auth('orContinueWith')}
            </span>
          </div>
        </div>

        <Button variant="outline" className="w-full" type="button" disabled>
          GitHub
        </Button>
      </CardContent>
      <CardFooter className="px-0 flex flex-wrap items-center justify-center gap-1.5 text-sm text-muted-foreground mt-2">
        {t.auth('noAccount')}
        <Link
          href="/register"
          className="font-medium text-primary hover:underline hover:text-primary-deep transition-colors"
        >
          {t.auth('signUp')}
        </Link>
      </CardFooter>
    </Card>
  );
}
