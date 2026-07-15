import { describe, expect, it } from 'vitest';

import {
  ApiErrorCode,
  ClientErrorCode,
  HttpStatusErrorCodeMap,
  normalizeLegacyErrorCode,
} from '@/http/codes';

const apiErrorCodePattern =
  /^(?:COMMON|AUTH|USER|API_KEY|ORGANIZATION|PERMISSION|ROLE|NOTIFICATION)(?:\.[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*)+$/;
const clientErrorCodePattern = /^CLIENT\.[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)*$/;
const legacyErrorCodePattern = /^(?:SYS|AUTH|BIZ|VAL)_\d{3}$/;

function duplicateValues(values: string[]): string[] {
  const seen = new Set<string>();
  const duplicates = new Set<string>();

  values.forEach((value) => {
    if (seen.has(value)) {
      duplicates.add(value);
    }

    seen.add(value);
  });

  return Array.from(duplicates).sort();
}

describe('error code vocabulary', () => {
  it('keeps API error codes on canonical server namespaces', () => {
    const values = Object.values(ApiErrorCode);
    const offenders = values.filter((value) => !apiErrorCodePattern.test(value));

    expect(offenders).toEqual([]);
    expect(duplicateValues(values)).toEqual([]);
  });

  it('keeps client fallback codes separate from server error_code values', () => {
    const values = Object.values(ClientErrorCode);
    const offenders = values.filter((value) => !clientErrorCodePattern.test(value));

    expect(offenders).toEqual([]);
    expect(duplicateValues(values)).toEqual([]);
  });

  it('maps HTTP status fallbacks to API error codes only', () => {
    const apiErrorCodes = new Set(Object.values(ApiErrorCode));
    const offenders = Object.entries(HttpStatusErrorCodeMap)
      .filter(([, errorCode]) => !apiErrorCodes.has(errorCode))
      .map(([status, errorCode]) => `${status}: ${errorCode}`);

    expect(offenders).toEqual([]);
  });

  it('keeps legacy underscore codes as normalization input only', () => {
    const legacyExamples = ['SYS_001', 'AUTH_401', 'VAL_400', 'BIZ_404'];
    const emittedCodes = [...Object.values(ApiErrorCode), ...Object.values(ClientErrorCode)];

    expect(emittedCodes.filter((code) => legacyErrorCodePattern.test(code))).toEqual([]);
    expect(legacyExamples.map(normalizeLegacyErrorCode)).toEqual([
      ClientErrorCode.UNKNOWN,
      ApiErrorCode.AUTH_UNAUTHORIZED,
      ApiErrorCode.COMMON_INVALID_INPUT,
      ApiErrorCode.COMMON_NOT_FOUND,
    ]);
  });
});
