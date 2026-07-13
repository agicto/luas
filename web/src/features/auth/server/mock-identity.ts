import 'server-only';

import type { AuthRuntimeMode } from '@/features/auth/server/auth-runtime';
import type { AuthUser } from '@/features/auth/types';

export interface MockLoginCredentials {
  email: string;
  password: string;
}

const mockLoginCredentials: MockLoginCredentials = {
  email: 'admin@example.com',
  password: 'admin123',
};

const mockUser: AuthUser = {
  id: 'demo-admin',
  email: mockLoginCredentials.email,
  name: 'Admin User',
  role: 'admin',
};

interface MockLoginPresentationContext {
  authMode: AuthRuntimeMode;
}

export function resolveMockLoginCredentials({
  authMode,
}: MockLoginPresentationContext): MockLoginCredentials | undefined {
  if (authMode !== 'mock-session') {
    return undefined;
  }

  return { ...mockLoginCredentials };
}

export function authenticateMockIdentity(
  email: string,
  password: string
): AuthUser | null {
  if (
    email !== mockLoginCredentials.email ||
    password !== mockLoginCredentials.password
  ) {
    return null;
  }

  return { ...mockUser };
}
