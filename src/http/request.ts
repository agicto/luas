import axios, { AxiosError, AxiosInstance, AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import type { ApiResponse as BaseApiResponse } from './types';

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
      // Network error handling
      if (!error.response) {
        return Promise.reject(new ApiError(
          'Network error, please check your connection', 
          'NETWORK_ERROR'
        ));
      }
      
      const { status, data } = error.response as AxiosResponse;
      
      // Format error message based on response
      let message = 'Request failed';
      let code: string | number = status;
      
      if (data && typeof data === 'object') {
        if (data.message) {
          message = data.message;
        } else if (data.error) {
          message = data.error;
        }
        
        if (data.code) {
          code = data.code;
        }
      }
      
      return Promise.reject(new ApiError(message, code, data, status));
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

