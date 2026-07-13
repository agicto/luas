export type ErrorHandlingOwner = 'global' | 'local';

export interface QueryOperationMeta extends Record<string, unknown> {
  errorHandling?: ErrorHandlingOwner;
}

export const LOCAL_ERROR_HANDLING_META = {
  errorHandling: 'local',
} as const satisfies QueryOperationMeta;

export function hasLocalErrorHandling(
  meta: QueryOperationMeta | undefined
): boolean {
  return meta?.errorHandling === 'local';
}
