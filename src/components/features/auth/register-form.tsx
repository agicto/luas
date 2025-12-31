'use client';

import { useState } from 'react';
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
import { Checkbox } from '@/components/ui/checkbox';
import { Icons } from '@/components/ui/icons';
import { useAuthStore, authSelectors } from '@/store/auth-store';
import { useRegister } from '@/hooks/use-auth';

// Form validation schema
const registerSchema = z.object({
  name: z
    .string()
    .min(1, 'Name is required')
    .min(2, 'Name must be at least 2 characters')
    .max(50, 'Name is too long'),
  email: z
    .string()
    .min(1, 'Email is required')
    .email('Please enter a valid email address'),
  password: z
    .string()
    .min(8, 'Password must be at least 8 characters')
    .max(100, 'Password is too long')
    .regex(
      /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[@$!%*?&])[A-Za-z\d@$!%*?&]/,
      'Password must contain at least one uppercase letter, one lowercase letter, one number and one special character'
    ),
  confirmPassword: z
    .string()
    .min(1, 'Please confirm your password'),
  terms: z
    .boolean()
    .refine(val => val === true, 'You must accept the terms and conditions'),
}).refine(data => data.password === data.confirmPassword, {
  message: 'Passwords do not match',
  path: ['confirmPassword'],
});

type RegisterFormData = z.infer<typeof registerSchema>;

interface RegisterFormProps {
  className?: string;
}

export function RegisterForm({ className }: RegisterFormProps) {
  const router = useRouter();
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  
  // Auth state
  const { mutateAsync: register, isPending: isMutationLoading } = useRegister();
  const systemFeatures = useAuthStore.use.systemFeatures();
  
  // Derived state - pass full state to selectors
  const canRegister = systemFeatures ? authSelectors.canRegister(useAuthStore.getState()) : false;
  const hasSocialLogin = systemFeatures ? authSelectors.hasSocialLogin(useAuthStore.getState()) : false;
  
  // Form setup
  const {
    register: registerField,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      name: '',
      email: '',
      password: '',
      confirmPassword: '',
      terms: false,
    },
  });

  // Form submission
  const onSubmit = async (data: RegisterFormData) => {
    try {
      await register(data);
    } catch (err) {
      // Error handled by hook's toast
      console.error('Registration submission error:', err);
    }
  };

  const isFormLoading = isMutationLoading || isSubmitting;

  // Show message if registration is not allowed
  if (systemFeatures && !canRegister) {
    return (
      <div className={cn('flex flex-col gap-6', className)}>
        <Card>
          <CardHeader className="text-center">
            <CardTitle className="text-2xl font-bold">Registration Disabled</CardTitle>
          </CardHeader>
          
          <CardContent className="space-y-6">
            <Alert>
              <Icons.AlertCircle className="h-4 w-4" />
              <AlertDescription>
                Registration is currently disabled for this platform. Please contact your administrator for access.
              </AlertDescription>
            </Alert>
            
            <div className="text-center">
              <Link
                href="/login"
                className="font-medium text-primary hover:underline"
              >
                Back to sign in
              </Link>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className={cn('flex flex-col gap-6', className)}>
      <Card>
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold">Create account</CardTitle>
          <p className="text-muted-foreground">
            Get started with the platform
          </p>
        </CardHeader>
        
        <CardContent className="space-y-6">
          {/* Error Alert removed - now using toast */}

          {/* Registration Form */}
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {/* Name Field */}
            <div className="space-y-2">
              <Label htmlFor="name">Full Name</Label>
              <Input
                id="name"
                type="text"
                placeholder="Enter your full name"
                autoComplete="name"
                disabled={isFormLoading}
                {...registerField('name')}
                aria-invalid={errors.name ? 'true' : 'false'}
              />
              {errors.name && (
                <p className="text-sm text-destructive">{errors.name.message}</p>
              )}
            </div>

            {/* Email Field */}
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                placeholder="Enter your email"
                autoComplete="email"
                disabled={isFormLoading}
                {...registerField('email')}
                aria-invalid={errors.email ? 'true' : 'false'}
              />
              {errors.email && (
                <p className="text-sm text-destructive">{errors.email.message}</p>
              )}
            </div>

            {/* Password Field */}
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Create a strong password"
                  autoComplete="new-password"
                  disabled={isFormLoading}
                  {...registerField('password')}
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

            {/* Confirm Password Field */}
            <div className="space-y-2">
              <Label htmlFor="confirmPassword">Confirm Password</Label>
              <div className="relative">
                <Input
                  id="confirmPassword"
                  type={showConfirmPassword ? 'text' : 'password'}
                  placeholder="Confirm your password"
                  autoComplete="new-password"
                  disabled={isFormLoading}
                  {...registerField('confirmPassword')}
                  aria-invalid={errors.confirmPassword ? 'true' : 'false'}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                  onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                  disabled={isFormLoading}
                  tabIndex={-1}
                >
                  {showConfirmPassword ? (
                    <Icons.EyeOff className="h-4 w-4" />
                  ) : (
                    <Icons.Eye className="h-4 w-4" />
                  )}
                  <span className="sr-only">
                    {showConfirmPassword ? 'Hide password' : 'Show password'}
                  </span>
                </Button>
              </div>
              {errors.confirmPassword && (
                <p className="text-sm text-destructive">{errors.confirmPassword.message}</p>
              )}
            </div>

            {/* Terms and Conditions */}
            <div className="flex items-start space-x-2">
              <Checkbox
                id="terms"
                disabled={isFormLoading}
                {...registerField('terms')}
                aria-invalid={errors.terms ? 'true' : 'false'}
              />
              <div className="grid gap-1.5 leading-none">
                <Label htmlFor="terms" className="text-sm font-normal">
                  I agree to the{' '}
                  <Link
                    href="/terms"
                    className="underline hover:text-foreground"
                    target="_blank"
                  >
                    Terms of Service
                  </Link>{' '}
                  and{' '}
                  <Link
                    href="/privacy"
                    className="underline hover:text-foreground"
                    target="_blank"
                  >
                    Privacy Policy
                  </Link>
                </Label>
                {errors.terms && (
                  <p className="text-sm text-destructive">{errors.terms.message}</p>
                )}
              </div>
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
              Create account
            </Button>
          </form>

          {/* Social Registration */}
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
                    console.log('Google registration');
                  }}
                >
                  <Icons.Google className="h-4 w-4" />
                  <span className="sr-only">Sign up with Google</span>
                </Button>
                
                <Button
                  variant="outline"
                  type="button"
                  disabled={isFormLoading}
                  onClick={() => {
                    // TODO: Implement Apple OAuth
                    console.log('Apple registration');
                  }}
                >
                  <Icons.Apple className="h-4 w-4" />
                  <span className="sr-only">Sign up with Apple</span>
                </Button>
                
                <Button
                  variant="outline"
                  type="button"
                  disabled={isFormLoading}
                  onClick={() => {
                    // TODO: Implement GitHub OAuth
                    console.log('GitHub registration');
                  }}
                >
                  <Icons.GitHub className="h-4 w-4" />
                  <span className="sr-only">Sign up with GitHub</span>
                </Button>
              </div>
            </>
          )}

          {/* Login Link */}
          <div className="text-center text-sm">
            Already have an account?{' '}
            <Link
              href="/login"
              className="font-medium text-primary hover:underline"
            >
              Sign in
            </Link>
          </div>
        </CardContent>
      </Card>

      {/* Password Requirements */}
      <div className="text-center text-xs text-muted-foreground">
        <div className="mb-2 font-medium">Password requirements:</div>
        <ul className="space-y-1 text-left max-w-sm mx-auto">
          <li>• At least 8 characters long</li>
          <li>• Contains uppercase and lowercase letters</li>
          <li>• Contains at least one number</li>
          <li>• Contains at least one special character</li>
        </ul>
      </div>
    </div>
  );
}
