import axios, { AxiosError, AxiosInstance, AxiosRequestConfig } from 'axios';
import { handleError } from './error-handler';
import { getAuthTokens } from '@/store/auth-store';
import { env } from '@/config/env';

/**
 * Custom Request Configuration
 */
export interface RequestConfig extends AxiosRequestConfig {
  skipAuth?: boolean;
  skipErrorHandler?: boolean;
}

/**
 * ApiError Class to encapsulate API-related errors
 */
export class ApiError extends Error {
  code: string | number;
  status?: number;
  constructor(message: string, code: string | number, status?: number) {
    super(message);
    this.code = code;
    this.status = status;
  }
}

/**
 * HttpClient provides a consistent interface for making HTTP requests.
 * It encapsulates axios instance management and interceptor logic.
 */
class HttpClient {
  private instance: AxiosInstance;

  constructor(config: RequestConfig) {
    this.instance = axios.create({
      timeout: 30000,
      ...config,
    });

    this.setupInterceptors();
  }

  private setupInterceptors() {
    // Request Interceptor: Auth Handling
    this.instance.interceptors.request.use(
      (config) => {
        const { skipAuth } = config as RequestConfig;
        if (!skipAuth) {
          const { accessToken } = getAuthTokens();
          if (accessToken) {
            config.headers.Authorization = `Bearer ${accessToken}`;
          }
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response Interceptor: Data Extraction & Error Handling
    this.instance.interceptors.response.use(
      (response) => {
        const { data } = response;
        // Standard payload extraction (assuming { code, data, message } format)
        return data && typeof data === 'object' && 'data' in data ? data.data : data;
      },
      async (error: AxiosError) => {
        const originalRequest = error.config as RequestConfig;
        const body = error.response?.data as any;

        const apiError = new ApiError(
          body?.message || body?.error || error.message,
          body?.code || 'FETCH_ERROR',
          error.response?.status
        );

        if (!originalRequest?.skipErrorHandler) {
          handleError(apiError);
        }

        return Promise.reject(apiError);
      }
    );
  }

  // Pure promise-based methods
  public get<T = any>(url: string, config?: RequestConfig): Promise<T> {
    return this.instance.get(url, config);
  }

  public post<T = any>(url: string, data?: any, config?: RequestConfig): Promise<T> {
    return this.instance.post(url, data, config);
  }

  public put<T = any>(url: string, data?: any, config?: RequestConfig): Promise<T> {
    return this.instance.put(url, data, config);
  }

  public patch<T = any>(url: string, data?: any, config?: RequestConfig): Promise<T> {
    return this.instance.patch(url, data, config);
  }

  public delete<T = any>(url: string, config?: RequestConfig): Promise<T> {
    return this.instance.delete(url, config);
  }
}

/**
 * Factory function to create new request instances
 */
export const createRequest = (config: RequestConfig = {}) => {
  return new HttpClient(config);
};

/**
 * Default instance for the primary API
 */
export const request = createRequest({
  baseURL: env.NEXT_PUBLIC_API_URL,
});

export default request;
