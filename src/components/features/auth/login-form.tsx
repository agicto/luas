'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import * as z from 'zod';

import { cn } from '@/utils';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Icons } from '@/components/ui/icons';
import { useLogin } from '@/hooks/use-auth';
import { env } from '@/config/env';
import { useAuthStore, authSelectors } from '@/store/auth-store';

const loginSchema = z.object({
  email: z.string().min(1, 'Required').email('Invalid email'),
  password: z.string().min(6, 'Too short'),
});

type LoginFormData = z.infer<typeof loginSchema>;

export function LoginForm({ className }: { className?: string }) {
  const [showPassword, setShowPassword] = useState(false);
  const { mutateAsync: login, isPending: isMutationLoading } = useLogin();
  const systemFeatures = useAuthStore.use.systemFeatures();

  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
  });

  const onSubmit = async (data: LoginFormData) => {
    try { await login({ ...data, remember: false }); } catch (err) { console.error(err); }
  };

  const isFormLoading = isMutationLoading || isSubmitting;
  const hasSocial = systemFeatures ? authSelectors.hasSocialLogin(useAuthStore.getState()) : false;

  return (
    <div className={cn('flex flex-col gap-6', className)}>
      <Card>
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold">Welcome back</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" disabled={isFormLoading} {...register('email')} />
              {errors.email && <p className="text-sm text-destructive">{errors.email.message}</p>}
            </div>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="password">Password</Label>
                <Link href="/forgot-password" className="text-primary text-sm">Forgot password?</Link>
              </div>
              <div className="relative">
                <Input id="password" type={showPassword ? 'text' : 'password'} disabled={isFormLoading} {...register('password')} />
                <Button type="button" variant="ghost" size="sm" className="absolute right-0 top-0 h-full px-3" onClick={() => setShowPassword(!showPassword)}>
                  {showPassword ? <Icons.EyeOff className="h-4 w-4" /> : <Icons.Eye className="h-4 w-4" />}
                </Button>
              </div>
            </div>
            <Button type="submit" className="w-full" disabled={isFormLoading}>
              {isFormLoading && <Icons.Spinner className="mr-2 h-4 w-4 animate-spin" />}
              Sign in
            </Button>
          </form>
          {hasSocial && (
            <div className="grid grid-cols-3 gap-3">
              <Button variant="outline"><Icons.Google className="h-4 w-4" /></Button>
              <Button variant="outline"><Icons.Apple className="h-4 w-4" /></Button>
              <Button variant="outline"><Icons.GitHub className="h-4 w-4" /></Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
