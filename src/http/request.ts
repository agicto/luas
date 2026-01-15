import axios, { AxiosError, AxiosInstance, AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import type { ApiResponse as BaseApiResponse } from './types';
import { handleError } from './error-handler';
import { getAuthTokens, setAuthTokens } from '@/store/auth-store';
import { publicEnv, serverEnv } from '@/config/env';

// Environment configuration
const API_URL =
  typeof window === 'undefined'
    ? (serverEnv.INTERNAL_API_URL || publicEnv.NEXT_PUBLIC_API_URL || 'http://localhost:3000/api')
    : (publicEnv.NEXT_PUBLIC_API_URL || '/api');
const API_TIMEOUT = publicEnv.NEXT_PUBLIC_API_TIMEOUT;

// Request configuration type
export interface RequestConfig extends AxiosRequestConfig {
  skipAuth?: boolean;
  skipErrorHandler?: boolean;
  baseURL?: string;
  _retry?: boolean; // Internal flag for silent refresh
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

// Token refresh state
let isRefreshing = false;
let failedQueue: any[] = [];

const processQueue = (error: any, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

/**
 * Modular Interceptors
 */
const interceptors = {
  request: {
    // 1. Auth Interceptor: Inject Bearer Token
    auth: (config: InternalAxiosRequestConfig & RequestConfig) => {
      if (!config.skipAuth) {
        const { accessToken } = getAuthTokens();
        if (accessToken) {
          config.headers.Authorization = `Bearer ${accessToken}`;
        }
      }
      
      if (!config.headers['Content-Type']) {
        config.headers['Content-Type'] = 'application/json';
      }
      return config;
    },
    error: (error: AxiosError) => Promise.reject(error),
  },
  
  response: {
    // 1. Success Interceptor: Unwrap data
    success: (response: AxiosResponse) => {
      const { data } = response;
      if (data && typeof data === 'object' && 'data' in data) {
        return (data as { data: unknown }).data;
      }
      return data;
    },
    
    // 2. Error Interceptor: Handle errors and Silent Refresh
    error: async (error: AxiosError) => {
      const originalRequest = error.config as InternalAxiosRequestConfig & RequestConfig;
      
      // Handle 401 Unauthorized - Silent Refresh logic
      if (error.response?.status === 401 && !originalRequest._retry && !originalRequest.skipAuth) {
        if (isRefreshing) {
          return new Promise((resolve, reject) => {
            failedQueue.push({ resolve, reject });
          })
            .then((token) => {
              originalRequest.headers.Authorization = `Bearer ${token}`;
              return axios(originalRequest);
            })
            .catch((err) => Promise.reject(err));
        }

        originalRequest._retry = true;
        isRefreshing = true;

        const { refreshToken } = getAuthTokens();
        if (refreshToken) {
          try {
            // Using a clean axios instance to avoid interceptor recursion
            const response = await axios.post(`${API_URL}/auth/refresh`, { refreshToken });
            const { accessToken: newAccessToken, refreshToken: newRefreshToken } = response.data.data || response.data;
            
            setAuthTokens(newAccessToken, newRefreshToken);
            processQueue(null, newAccessToken);
            
            originalRequest.headers.Authorization = `Bearer ${newAccessToken}`;
            return axios(originalRequest);
          } catch (refreshError) {
            processQueue(refreshError, null);
            // Optional: Logout user or redirect to login
            setAuthTokens(null, null);
            return Promise.reject(refreshError);
          } finally {
            isRefreshing = false;
          }
        }
      }

      // Format and normalize error
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

      const config = error.config as RequestConfig;
      if (!config?.skipErrorHandler) {
        handleError(apiError);
      }
      
      return Promise.reject(apiError);
    },
  },
};

// Interceptor setup function
function setupInterceptors(instance: AxiosInstance) {
  instance.interceptors.request.use(interceptors.request.auth, interceptors.request.error);
  instance.interceptors.response.use(interceptors.response.success, interceptors.response.error);
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

// Cache for custom baseURL instances
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

export const requestUtils = {
  createInstance: createAxiosInstance,
};

export type ApiResponse = BaseApiResponse;

export default request;

