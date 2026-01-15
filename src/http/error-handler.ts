import { toast } from 'sonner';
import { ApiError } from './request';
import { env } from '@/config/env';

/**
 * Global Error Handler Configuration
 */
export interface ErrorHandlerConfig {
  silent?: boolean;
  notify?: boolean;
  fallbackMessage?: string;
}

const DEFAULT_CONFIG: ErrorHandlerConfig = {
  silent: false,
  notify: true,
  fallbackMessage: 'An unexpected error occurred',
};

/**
 * Centralized error handler for API and application errors.
 */
export function handleError(error: unknown, config: ErrorHandlerConfig = {}): void {
  const mergedConfig = { ...DEFAULT_CONFIG, ...config };
  
  if (mergedConfig.silent) return;

  let message = mergedConfig.fallbackMessage || 'Error';
  let errorCode: string | number | undefined;

  if (error instanceof ApiError) {
    message = error.message;
    errorCode = error.code;
    
    switch (error.status) {
      case 403: message = 'Permission denied'; break;
      case 404: message = 'Resource not found'; break;
      case 500: message = 'Server error'; break;
    }
  } else if (error instanceof Error) {
    message = error.message;
  }

  if (mergedConfig.notify) {
    toast.error(message, {
      description: errorCode ? `Code: ${errorCode}` : undefined,
    });
  }

  if (env.NODE_ENV === 'development') {
    console.error('[GlobalErrorHandler]', error);
  }
}
