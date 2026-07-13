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
import { Checkbox } from '@/components/ui/checkbox';
import { useRegister } from '@/features/auth/hooks/use-auth';
import { hasAuthFieldError, resolveAuthErrorKey } from '@/features/auth/utils/auth-error';
import { useT } from '@/i18n';

/** Registration entry point with localized, contract-aware mutation feedback. */
export function RegisterForm() {
  const t = useT();
  const { mutate: register, isPending, error, reset } = useRegister();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [acceptedTerms, setAcceptedTerms] = useState(false);
  const nameError = hasAuthFieldError(error, 'name') ? t.auth('nameInvalid') : undefined;
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
    if (!acceptedTerms) {
      return;
    }

    register({ name, email, password });
  };

  return (
    <Card className="border-none shadow-none bg-transparent">
      <CardHeader className="space-y-1 px-0">
        <CardTitle className="text-2xl font-bold tracking-tight">
          {t.auth('createAccount')}
        </CardTitle>
        <CardDescription>{t.auth('getStarted')}</CardDescription>
      </CardHeader>
      <CardContent className="px-0 pt-4">
        <form onSubmit={handleSubmit} className="space-y-4" aria-busy={isPending}>
          {error && (
            <Alert variant="destructive">
              <AlertCircle aria-hidden="true" className="size-4" />
              <AlertDescription>{t(resolveAuthErrorKey(error, 'register'))}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="name">{t.auth('fullName')}</Label>
            <Input
              id="name"
              name="name"
              autoComplete="name"
              placeholder={t.auth('enterFullName')}
              required
              disabled={isPending}
              className="bg-bg-canvas"
              value={name}
              errorText={nameError}
              onChange={event => {
                setName(event.target.value);
                clearMutationError();
              }}
            />
          </div>
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
            <Label htmlFor="password">{t.auth('password')}</Label>
            <Input
              id="password"
              name="password"
              type="password"
              autoComplete="new-password"
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

          <div className="flex items-start space-x-2 pt-2">
            <Checkbox
              id="terms"
              name="terms"
              checked={acceptedTerms}
              disabled={isPending}
              onCheckedChange={checked => {
                setAcceptedTerms(Boolean(checked));
                clearMutationError();
              }}
              required
            />
            <div className="grid gap-1.5 leading-none">
              <label
                htmlFor="terms"
                className="text-xs font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                {t.auth('agreeToTerms')} {t.auth('termsOfService')} {t.auth('and')}{' '}
                {t.auth('privacyPolicy')}
              </label>
            </div>
          </div>

          <Button
            type="submit"
            className="w-full font-semibold"
            disabled={!acceptedTerms}
            loading={isPending}
          >
            {t.auth('signUp')}
          </Button>
        </form>
      </CardContent>
      <CardFooter className="px-0 flex flex-wrap items-center justify-center gap-1.5 text-sm text-muted-foreground mt-2">
        {t.auth('hasAccount')}
        <Link
          href="/login"
          className="font-medium text-primary hover:underline hover:text-primary-deep transition-colors"
        >
          {t.auth('signIn')}
        </Link>
      </CardFooter>
    </Card>
  );
}
