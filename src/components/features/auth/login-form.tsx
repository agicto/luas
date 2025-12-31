'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import * as z from 'zod';

import { cn } from '@/utils';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Icons } from '@/components/ui/icons';
import { useAuthStore, authSelectors } from '@/store/auth-store';
import { useLogin } from '@/hooks/use-auth';

// Form validation schema
const loginSchema = z.object({
  email: z
    .string()
    .min(1, 'Email is required')
    .email('Please enter a valid email address'),
  password: z
    .string()
    .min(6, 'Password must be at least 6 characters')
    .max(100, 'Password is too long'),
  remember: z.boolean().optional(),
});

type LoginFormData = z.infer<typeof loginSchema>;

interface LoginFormProps {
  className?: string;
}

export function LoginForm({ className }: LoginFormProps) {
  const router = useRouter();
  const [showPassword, setShowPassword] = useState(false);
  
  // Auth state
  const { mutateAsync: login, isPending: isMutationLoading } = useLogin();
  const systemFeatures = useAuthStore.use.systemFeatures();

  // Log mock credentials in development if mock auth is enabled
  useEffect(() => {
    if (process.env.NODE_ENV === 'development' && process.env.NEXT_PUBLIC_USE_MOCK_AUTH === 'true') {
      console.log('🚀 [Mock Auth] Available accounts:');
      console.table([
        { email: 'admin@example.com', password: 'admin123', role: 'admin' },
        { email: 'user@example.com', password: 'user123', role: 'user' },
      ]);
    }
  }, []);
  
  // Derived state - pass full state to selectors
  const canRegister = systemFeatures ? authSelectors.canRegister(useAuthStore.getState()) : false;
  const hasSocialLogin = systemFeatures ? authSelectors.hasSocialLogin(useAuthStore.getState()) : false;
  
  // Form setup
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: '',
      password: '',
      remember: false,
    },
  });

  // Form submission
  const onSubmit = async (data: LoginFormData) => {
    try {
      await login(data);
    } catch (err) {
      // Error handled by hook's toast
      console.error('Login submission error:', err);
    }
  };

  const isFormLoading = isMutationLoading || isSubmitting;

  return (
    <div className={cn('flex flex-col gap-6', className)}>
      <Card>
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold">Welcome back</CardTitle>
          <p className="text-muted-foreground">
            Sign in to your account
          </p>
        </CardHeader>
        
        <CardContent className="space-y-6">
          {/* Error Alert removed - now using toast */}

          {/* Login Form */}
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {/* Email Field */}
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                placeholder="Enter your email"
                autoComplete="email"
                disabled={isFormLoading}
                {...register('email')}
                aria-invalid={errors.email ? 'true' : 'false'}
              />
              {errors.email && (
                <p className="text-sm text-destructive">{errors.email.message}</p>
              )}
            </div>

            {/* Password Field */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="password">Password</Label>
                <Link
                  href="/forgot-password"
                  className="text-sm text-primary hover:underline"
                  tabIndex={-1}
                >
                  Forgot password?
                </Link>
              </div>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Enter your password"
                  autoComplete="current-password"
                  disabled={isFormLoading}
                  {...register('password')}
                  aria-invalid={errors.password ? 'true' : 'false'}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                  onClick={() => setShowPassword(!showPassword)}
                  disabled={isFormLoading}
                  tabIndex={-1}
                >
                  {showPassword ? (
                    <Icons.EyeOff className="h-4 w-4" />
                  ) : (
                    <Icons.Eye className="h-4 w-4" />
                  )}
                  <span className="sr-only">
                    {showPassword ? 'Hide password' : 'Show password'}
                  </span>
                </Button>
              </div>
              {errors.password && (
                <p className="text-sm text-destructive">{errors.password.message}</p>
              )}
            </div>

            {/* Remember Me */}
            <div className="flex items-center space-x-2">
              <input
                id="remember"
                type="checkbox"
                className="rounded border-gray-300"
                disabled={isFormLoading}
                {...register('remember')}
              />
              <Label htmlFor="remember" className="text-sm font-normal">
                Remember me for 30 days
              </Label>
            </div>

            {/* Submit Button */}
            <Button
              type="submit"
              className="w-full"
              disabled={isFormLoading}
            >
              {isFormLoading && (
                <Icons.Spinner className="mr-2 h-4 w-4 animate-spin" />
              )}
              Sign in
            </Button>
          </form>

          {/* Social Login */}
          {hasSocialLogin && (
            <>
              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-background px-2 text-muted-foreground">
                    Or continue with
                  </span>
                </div>
              </div>

              <div className="grid grid-cols-3 gap-3">
                <Button
                  variant="outline"
                  type="button"
                  disabled={isFormLoading}
                  onClick={() => {
                    // TODO: Implement Google OAuth
                    console.log('Google login');
                  }}
                >
                  <Icons.Google className="h-4 w-4" />
                  <span className="sr-only">Sign in with Google</span>
                </Button>
                
                <Button
                  variant="outline"
                  type="button"
                  disabled={isFormLoading}
                  onClick={() => {
                    // TODO: Implement Apple OAuth
                    console.log('Apple login');
                  }}
                >
                  <Icons.Apple className="h-4 w-4" />
                  <span className="sr-only">Sign in with Apple</span>
                </Button>
                
                <Button
                  variant="outline"
                  type="button"
                  disabled={isFormLoading}
                  onClick={() => {
                    // TODO: Implement GitHub OAuth
                    console.log('GitHub login');
                  }}
                >
                  <Icons.GitHub className="h-4 w-4" />
                  <span className="sr-only">Sign in with GitHub</span>
                </Button>
              </div>
            </>
          )}

          {/* Registration Link */}
          {canRegister && (
            <div className="text-center text-sm">
              Don&apos;t have an account?{' '}
              <Link
                href="/register"
                className="font-medium text-primary hover:underline"
              >
                Create account
              </Link>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Terms and Privacy */}
      <div className="text-center text-xs text-muted-foreground">
        By signing in, you agree to our{' '}
        <Link href="/terms" className="underline hover:text-foreground">
          Terms of Service
        </Link>{' '}
        and{' '}
        <Link href="/privacy" className="underline hover:text-foreground">
          Privacy Policy
        </Link>
        .
      </div>
    </div>
  );
}
