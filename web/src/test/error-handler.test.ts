import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiErrorCode } from '@/http/codes';
import { handleError } from '@/http/error-handler';
import { ApiError } from '@/http/request';

const toast = vi.hoisted(() => ({ error: vi.fn() }));

vi.mock('sonner', () => ({ toast }));

describe('global error presentation', () => {
  beforeEach(() => {
    toast.error.mockReset();
  });

  it('does not expose backend error detail in the global fallback', () => {
    handleError(
      new ApiError(
        'sensitive backend validation detail',
        ApiErrorCode.COMMON_VALIDATION_FAILED,
        422
      )
    );

    expect(toast.error).toHaveBeenCalledWith(
      'An unexpected error occurred',
      {
        description: `Error Code: ${ApiErrorCode.COMMON_VALIDATION_FAILED}`,
      }
    );
    expect(toast.error).not.toHaveBeenCalledWith(
      expect.stringContaining('sensitive backend validation detail'),
      expect.anything()
    );
  });

  it('allows a reviewed local fallback to replace the generic copy', () => {
    handleError(new Error('sensitive runtime detail'), {
      fallbackMessage: 'Unable to save changes',
    });

    expect(toast.error).toHaveBeenCalledWith('Unable to save changes', {
      description: undefined,
    });
  });
});
