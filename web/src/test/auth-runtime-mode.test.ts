import { describe, expect, it } from 'vitest';

import { resolveAuthRuntimeMode } from '@/features/auth/server/auth-runtime';

describe('auth runtime mode', () => {
  it('uses server-verified mock sessions for local development', () => {
    expect(
      resolveAuthRuntimeMode({
        apiUrl: '/api',
        appUrl: 'http://localhost:3000',
        authAdapterEnabled: false,
        mockBffEnabled: false,
        nodeEnv: 'development',
      })
    ).toBe('mock-session');
  });

  it('uses server-verified mock sessions for explicit demo deployments', () => {
    expect(
      resolveAuthRuntimeMode({
        apiUrl: 'https://demo.example.com/api/',
        appUrl: 'https://demo.example.com',
        authAdapterEnabled: false,
        mockBffEnabled: true,
        nodeEnv: 'production',
      })
    ).toBe('mock-session');
  });

  it('defers to the browser for an external real API', () => {
    expect(
      resolveAuthRuntimeMode({
        apiUrl: 'https://api.example.com',
        appUrl: 'https://app.example.com',
        authAdapterEnabled: false,
        mockBffEnabled: false,
        nodeEnv: 'production',
      })
    ).toBe('client-session');
  });

  it('does not mistake a production same-origin proxy for the mock BFF', () => {
    expect(
      resolveAuthRuntimeMode({
        apiUrl: '/api',
        appUrl: 'https://app.example.com',
        authAdapterEnabled: false,
        mockBffEnabled: false,
        nodeEnv: 'production',
      })
    ).toBe('client-session');
  });

  it('does not use mock sessions when development points at a real API', () => {
    expect(
      resolveAuthRuntimeMode({
        apiUrl: 'https://api.example.com',
        appUrl: 'http://localhost:3000',
        authAdapterEnabled: false,
        mockBffEnabled: false,
        nodeEnv: 'development',
      })
    ).toBe('client-session');
  });

  it('uses the server-resolved API session when the production adapter owns /api', () => {
    expect(
      resolveAuthRuntimeMode({
        apiUrl: '/api',
        appUrl: 'https://app.example.com',
        authAdapterEnabled: true,
        mockBffEnabled: false,
        nodeEnv: 'production',
      })
    ).toBe('api-session');
  });

  it('gives the production adapter precedence over an explicit mock opt-in', () => {
    expect(
      resolveAuthRuntimeMode({
        apiUrl: '/api',
        appUrl: 'https://app.example.com',
        authAdapterEnabled: true,
        mockBffEnabled: true,
        nodeEnv: 'production',
      })
    ).toBe('api-session');
  });
});
