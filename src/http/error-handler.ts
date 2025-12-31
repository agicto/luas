import { toast } from 'sonner';
import { ApiError } from './request';

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
 * 
 * @param error The error object to handle
 * @param config Optional configuration for specific request
 */
export function handleError(error: unknown, config: ErrorHandlerConfig = {}): void {
  const mergedConfig = { ...DEFAULT_CONFIG, ...config };
  
  if (mergedConfig.silent) return;

  let message = mergedConfig.fallbackMessage || 'Error';
  let errorCode: string | number | undefined;

  if (error instanceof ApiError) {
    message = error.message;
    errorCode = error.code;
    
    // Handle specific status codes
    switch (error.status) {
      case 401:
        // Unauthorized logic could go here (e.g., redirect to login)
        // We might want to use an event bus or a store to handle this
        console.warn('[ErrorHandler] Unauthorized access detected');
        break;
      case 403:
        message = 'You do not have permission to perform this action';
        break;
      case 404:
        message = 'Resource not found';
        break;
      case 500:
        message = 'Server error, please try again later';
        break;
    }
  } else if (error instanceof Error) {
    message = error.message;
  } else if (typeof error === 'string') {
    message = error;
  }

  // Notify user via Toast
  if (mergedConfig.notify) {
    toast.error(message, {
      description: errorCode ? `Error Code: ${errorCode}` : undefined,
    });
  }

  // Log to console in development
  if (process.env.NODE_ENV === 'development') {
    console.error('[GlobalErrorHandler]', {
      message,
      errorCode,
      originalError: error,
    });
  }
}
