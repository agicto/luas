'use client';

import { useState } from 'react';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import { useRegister } from '@/features/auth/hooks/use-auth';
import { useT } from '@/i18n';

/**
 * RegisterForm - Pure UI Component
 * 
 * A high-quality sign-up form UI without business logic.
 * Use this as a starting point to implement your registration flow.
 */
export function RegisterForm() {
  const t = useT();
  const { mutate: register, isPending } = useRegister();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [acceptedTerms, setAcceptedTerms] = useState(false);

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
        <CardDescription>
          {t.auth('getStarted')}
        </CardDescription>
      </CardHeader>
      <CardContent className="px-0 pt-4">
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">{t.auth('fullName')}</Label>
            <Input 
              id="name" 
              placeholder={t.auth('enterFullName')}
              required 
              className="bg-bg-canvas"
              value={name}
              onChange={(event) => setName(event.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="email">{t.auth('email')}</Label>
            <Input 
              id="email" 
              type="email" 
              placeholder={t.auth('enterEmail')}
              required 
              className="bg-bg-canvas"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">{t.auth('password')}</Label>
            <Input 
              id="password" 
              type="password" 
              required 
              className="bg-bg-canvas"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </div>
          
          <div className="flex items-start space-x-2 pt-2">
            <Checkbox
              id="terms"
              checked={acceptedTerms}
              onCheckedChange={(checked) => setAcceptedTerms(Boolean(checked))}
              required
            />
            <div className="grid gap-1.5 leading-none">
              <label
                htmlFor="terms"
                className="text-xs font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
              >
                {t.auth('agreeToTerms')} {t.auth('termsOfService')} {t.auth('and')} {t.auth('privacyPolicy')}
              </label>
            </div>
          </div>

          <Button type="submit" className="w-full font-semibold" disabled={isPending || !acceptedTerms}>
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
