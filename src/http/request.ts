import axios, { AxiosError, AxiosInstance, AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import type { ApiResponse as BaseApiResponse } from './types';
import { handleError } from './error-handler';

// Environment configuration
// In the browser, a relative baseURL works best for same-origin Route Handlers.
// On the server, axios requires an absolute URL, so allow overriding via INTERNAL_API_URL.
const API_URL =
  typeof window === 'undefined'
    ? (process.env.INTERNAL_API_URL || process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3000/api')
    : (process.env.NEXT_PUBLIC_API_URL || '/api');
const API_TIMEOUT = Number(process.env.NEXT_PUBLIC_API_TIMEOUT || 30000);

// Request configuration type
export interface RequestConfig extends AxiosRequestConfig {
  // Custom configuration options
  skipAuth?: boolean; // Skip authentication
  skipErrorHandler?: boolean; // Skip error handling
  baseURL?: string; // Optional custom baseURL
}

// Error type
export class ApiError extends Error {
  code: number | string;
  data?: unknown;
  status?: number;
  
  constructor(message: string, code: number | string, data?: unknown, status?: number) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.data = data;
    this.status = status;
  }
}

// Interceptor setup function
function setupInterceptors(instance: AxiosInstance) {
  // Request interceptor
  instance.interceptors.request.use(
    (config: InternalAxiosRequestConfig & RequestConfig) => {
      // Add default content type
      if (!config.headers['Content-Type']) {
        config.headers['Content-Type'] = 'application/json';
      }
      
      return config;
    },
    (error: AxiosError) => {
      return Promise.reject(error);
    }
  );

  // Response interceptor
  instance.interceptors.response.use(
    (response: AxiosResponse) => {
      const { data } = response;

      // If the backend uses a common envelope like { data: ... }, unwrap it.
      if (data && typeof data === 'object' && 'data' in data) {
        return (data as { data: unknown }).data;
      }

      return data;
    },
    (error: AxiosError) => {
      let message = 'Request failed';
      let code: string | number = 'UNKNOWN_ERROR';
      let data: unknown = undefined;
      let status: number | undefined = undefined;

      if (!error.response) {
        message = 'Network error, please check your connection';
        code = 'NETWORK_ERROR';
      } else {
        const response = error.response as AxiosResponse;
        status = response.status;
        data = response.data;
        code = status;

        if (data && typeof data === 'object') {
          const body = data as Record<string, any>;
          if (body.message) message = body.message;
          else if (body.error) message = body.error;
          if (body.code) code = body.code;
        }
      }

      const apiError = new ApiError(message, code, data, status);

      // Trigger global error handler unless skipped
      const config = error.config as RequestConfig;
      if (!config?.skipErrorHandler) {
        handleError(apiError);
      }
      
      return Promise.reject(apiError);
    }
  );
}

// Create axios instance factory
function createAxiosInstance(baseURL?: string): AxiosInstance {
  const inst = axios.create({
    baseURL: baseURL || API_URL,
    timeout: API_TIMEOUT,
    withCredentials: false,
    headers: {
      'Content-Type': 'application/json',
    },
  });
  setupInterceptors(inst);
  return inst;
}

// Default instance
const defaultInstance = createAxiosInstance();

// Cache for custom baseURL instances to avoid recreating on every request
const instanceCache = new Map<string, AxiosInstance>();

function getInstanceForBaseURL(baseURL?: string): AxiosInstance {
  if (!baseURL) return defaultInstance;

  let instance = instanceCache.get(baseURL);
  if (!instance) {
    instance = createAxiosInstance(baseURL);
    instanceCache.set(baseURL, instance);
  }
  return instance;
}

// Request method wrapper
export const request = {
  get: <T = unknown>(url: string, config?: RequestConfig): Promise<T> => {
    return getInstanceForBaseURL(config?.baseURL).get<unknown, T>(url, config);
  },

  post: <T = unknown>(url: string, data?: unknown, config?: RequestConfig): Promise<T> => {
    return getInstanceForBaseURL(config?.baseURL).post<unknown, T>(url, data, config);
  },

  put: <T = unknown>(url: string, data?: unknown, config?: RequestConfig): Promise<T> => {
    return getInstanceForBaseURL(config?.baseURL).put<unknown, T>(url, data, config);
  },

  delete: <T = unknown>(url: string, config?: RequestConfig): Promise<T> => {
    return getInstanceForBaseURL(config?.baseURL).delete<unknown, T>(url, config);
  },

  patch: <T = unknown>(url: string, data?: unknown, config?: RequestConfig): Promise<T> => {
    return getInstanceForBaseURL(config?.baseURL).patch<unknown, T>(url, data, config);
  },
};

// Export utilities for external use
export const requestUtils = {
  createInstance: createAxiosInstance,
};

// Re-export types for convenience
export type ApiResponse = BaseApiResponse;

export default request;

