import { beforeEach, describe, expect, it, vi } from 'vitest';

import { authService } from '@/features/auth/services/auth-service';
import type { AuthUser } from '@/features/auth/types';
import { ClientErrorCode } from '@/http/codes';

const request = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}));

vi.mock('@/http', () => ({ default: request }));

const user: AuthUser = {
  id: 'user-ada',
  email: 'ada@example.com',
  name: 'Ada Lovelace',
  role: 'admin',
};

describe('auth service response contracts', () => {
  beforeEach(() => {
    request.get.mockReset();
    request.post.mockReset();
  });

  it.each([
    ['login', () => authService.login({ email: 'ada@example.com', password: 'secret' })],
    [
      'register',
      () =>
        authService.register({
          name: 'Ada Lovelace',
          email: 'ada@example.com',
          password: 'secret-123',
        }),
    ],
  ])('rejects a malformed %s success payload', async (_name, invoke) => {
    request.post.mockResolvedValueOnce({});

    await expect(invoke()).rejects.toMatchObject({
      errorCode: ClientErrorCode.INVALID_RESPONSE,
    });
  });

  it('rejects a malformed current-session success payload', async () => {
    request.get.mockResolvedValueOnce({ user: { ...user, role: 'owner' } });

    await expect(authService.me()).rejects.toMatchObject({
      errorCode: ClientErrorCode.INVALID_RESPONSE,
    });
  });

  it('rejects a malformed logout success payload', async () => {
    request.post.mockResolvedValueOnce({ success: false });

    await expect(authService.logout()).rejects.toMatchObject({
      errorCode: ClientErrorCode.INVALID_RESPONSE,
    });
  });

  it('returns validated auth and logout payloads unchanged', async () => {
    request.post.mockResolvedValueOnce({ user }).mockResolvedValueOnce({ success: true });

    await expect(
      authService.login({ email: 'ada@example.com', password: 'secret' })
    ).resolves.toEqual({ user });
    await expect(authService.logout()).resolves.toEqual({ success: true });
  });
});
