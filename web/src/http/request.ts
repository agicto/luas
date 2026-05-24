import axios, { AxiosError, AxiosInstance, AxiosRequestConfig } from 'axios';
import { handleError } from './error-handler';
import { env } from '@/config/env';

/**
 * Custom Request Configuration
 */
export interface RequestConfig extends AxiosRequestConfig {
  skipErrorHandler?: boolean;
}

interface ApiErrorBody {
  code?: string | number;
  error?: string;
  message?: string;
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
      withCredentials: true,
      ...config,
    });

    this.setupInterceptors();
  }

  private setupInterceptors() {
    // Response Interceptor: Data Extraction & Error Handling
    this.instance.interceptors.response.use(
      (response) => {
        const { data } = response;
        // Standard payload extraction (assuming { code, data, message } format)
        return data && typeof data === 'object' && 'data' in data ? data.data : data;
      },
      async (error: AxiosError) => {
        const originalRequest = error.config as RequestConfig;
        const body = error.response?.data as ApiErrorBody | undefined;

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
  public get<T = unknown>(url: string, config?: RequestConfig): Promise<T> {
    return this.instance.get<T, T>(url, config);
  }

  public post<T = unknown, D = unknown>(url: string, data?: D, config?: RequestConfig): Promise<T> {
    return this.instance.post<T, T, D>(url, data, config);
  }

  public put<T = unknown, D = unknown>(url: string, data?: D, config?: RequestConfig): Promise<T> {
    return this.instance.put<T, T, D>(url, data, config);
  }

  public patch<T = unknown, D = unknown>(url: string, data?: D, config?: RequestConfig): Promise<T> {
    return this.instance.patch<T, T, D>(url, data, config);
  }

  public delete<T = unknown>(url: string, config?: RequestConfig): Promise<T> {
    return this.instance.delete<T, T>(url, config);
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
