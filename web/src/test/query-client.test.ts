import { describe, expect, it, vi } from 'vitest';

import { createQueryClient } from '@/config/query-client';
import { LOCAL_ERROR_HANDLING_META } from '@/config/query-meta';

const handleError = vi.hoisted(() => vi.fn());

vi.mock('@/http/error-handler', () => ({ handleError }));

async function executeFailingMutation(
  meta?: typeof LOCAL_ERROR_HANDLING_META
) {
  const client = createQueryClient();
  const mutationFn = vi.fn().mockRejectedValue(new Error('mutation failed'));
  const mutation = client.getMutationCache().build(client, {
    mutationFn,
    meta,
  });

  await expect(mutation.execute(undefined)).rejects.toThrow('mutation failed');

  return mutationFn;
}

describe('query client ownership defaults', () => {
  it('keeps provider instances isolated', () => {
    const first = createQueryClient();
    const second = createQueryClient();

    first.setQueryData(['session'], { user: 'ada' });

    expect(first.getQueryData(['session'])).toEqual({ user: 'ada' });
    expect(second.getQueryData(['session'])).toBeUndefined();
  });

  it('does not retry write mutations without endpoint-specific evidence', async () => {
    handleError.mockClear();

    const mutationFn = await executeFailingMutation();

    expect(mutationFn).toHaveBeenCalledTimes(1);
    expect(handleError).toHaveBeenCalledTimes(1);
  });

  it('does not duplicate locally owned error feedback', async () => {
    handleError.mockClear();

    const mutationFn = await executeFailingMutation(
      LOCAL_ERROR_HANDLING_META
    );

    expect(mutationFn).toHaveBeenCalledTimes(1);
    expect(handleError).not.toHaveBeenCalled();
  });
});
