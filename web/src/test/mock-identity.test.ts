import { describe, expect, it } from 'vitest';

import {
  authenticateMockIdentity,
  resolveMockLoginCredentials,
} from '@/features/auth/server/mock-identity';

describe('mock identity boundary', () => {
  it('exposes the preset only when the Web owns the mock session', () => {
    expect(
      resolveMockLoginCredentials({
        authMode: 'mock-session',
      })
    ).toEqual({
      email: 'admin@example.com',
      password: 'admin123',
    });

    expect(
      resolveMockLoginCredentials({
        authMode: 'client-session',
      })
    ).toBeUndefined();
  });

  it('returns a browser-safe user only for the exact mock credentials', () => {
    expect(authenticateMockIdentity('admin@example.com', 'admin123')).toEqual({
      id: 'demo-admin',
      email: 'admin@example.com',
      name: 'Admin User',
      role: 'admin',
    });
    expect(authenticateMockIdentity('admin@example.com', 'wrong')).toBeNull();
  });
});
