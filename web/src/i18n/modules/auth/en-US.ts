// Auth translations - English (US)
import type { AuthMessages } from './zh-Hans';

const messages: AuthMessages = {
  // Page titles
  welcomeBack: 'Welcome back',
  signInToContinue: 'Sign in to continue to your account',
  createAccount: 'Create account',
  getStarted: 'Get started with the platform',
  registrationDisabled: 'Registration Disabled',
  registrationDisabledMessage: 'Registration is currently disabled for this platform. Please contact your administrator for access.',
  backToSignIn: 'Back to sign in',
  
  // Form labels
  email: 'Email',
  password: 'Password',
  confirmPassword: 'Confirm Password',
  fullName: 'Full Name',
  
  // Form placeholders
  enterEmail: 'Enter your email',
  enterPassword: 'Enter your password',
  createPassword: 'Create a strong password',
  confirmYourPassword: 'Confirm your password',
  enterFullName: 'Enter your full name',
  
  // Actions
  login: 'Sign In',
  logout: 'Sign Out',
  register: 'Sign Up',
  signIn: 'Sign In',
  signUp: 'Sign Up',
  forgotPassword: 'Forgot password?',
  demoCredentials: 'Demo credentials',
  demoCredentialsHint: 'Use the preset login',
  demoCredentialsValue: 'admin@example.com / admin123',
  
  // Links
  noAccount: "Don't have an account?",
  hasAccount: 'Already have an account?',
  rememberMe: 'Remember me',
  
  // Terms
  agreeToTerms: 'I agree to the',
  termsOfService: 'Terms of Service',
  and: 'and',
  privacyPolicy: 'Privacy Policy',
  orContinueWith: 'Or continue with',
  
  // Password requirements
  passwordRequirements: 'Password requirements',
  passwordReqLength: 'At least 8 characters long',
  passwordReqCase: 'Contains uppercase and lowercase letters',
  passwordReqNumber: 'Contains at least one number',
  passwordReqSpecial: 'Contains at least one special character',
  
  // Decorative panel
  heroTitle: 'Build Amazing Products',
  heroSubtitle: 'The modern web scaffold for fast, reliable product delivery',
  feature1: 'Lightning-fast development',
  feature2: 'Enterprise-grade security',
  feature3: 'Scalable architecture',
  feature4: '24/7 global support',
  
  // Validation messages
  nameRequired: 'Name is required',
  nameMinLength: 'Name must be at least 2 characters',
  nameTooLong: 'Name is too long',
  emailRequired: 'Email is required',
  emailInvalid: 'Please enter a valid email address',
  passwordRequired: 'Password is required',
  passwordMinLength: 'Password must be at least 8 characters',
  passwordTooLong: 'Password is too long',
  passwordTooShort: 'Password is too short',
  passwordInvalid: 'Password must contain uppercase, lowercase, number and special character',
  confirmPasswordRequired: 'Please confirm your password',
  passwordsDoNotMatch: 'Passwords do not match',
  termsRequired: 'You must accept the terms and conditions',
  
  // Toast messages
  loginSuccess: 'Login successful',
  loginFailed: 'Login failed',
  welcomeBackUser: 'Welcome back, {name}',
  logoutSuccess: 'Logout successful',
  logoutFailed: 'Logout failed',
  registerSuccess: 'Registration successful',
  registerFailed: 'Registration failed',
  accountCreated: 'Account created, signing in...',
  invalidCredentials: 'Invalid email or password',
  
  // Social login
  signInWithGoogle: 'Sign in with Google',
  signInWithApple: 'Sign in with Apple',
  signInWithGithub: 'Sign in with GitHub',
  signUpWithGoogle: 'Sign up with Google',
  signUpWithApple: 'Sign up with Apple',
  signUpWithGithub: 'Sign up with GitHub',
  
  // Footer
  allRightsReserved: 'All rights reserved',
};

export default messages;
