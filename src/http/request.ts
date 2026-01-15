import axios, { AxiosError, AxiosInstance, AxiosResponse, InternalAxiosRequestConfig, AxiosRequestConfig } from 'axios';
import { handleError } from './error-handler';
import { getAuthTokens, setAuthTokens } from '@/store/auth-store';
import { env } from '@/config/env';

/**
 * Extreme Purification: Single instance, simple logic.
 */
const instance: AxiosInstance = axios.create({
  baseURL: env.NEXT_PUBLIC_API_URL,
  timeout: 30000,
});

export interface RequestConfig extends AxiosRequestConfig {
  skipAuth?: boolean;
  skipErrorHandler?: boolean;
  _retry?: boolean;
}

// Interceptors
instance.interceptors.request.use(
  (config) => {
    const { skipAuth } = config as RequestConfig;
    if (!skipAuth) {
      const { accessToken } = getAuthTokens();
      if (accessToken) config.headers.Authorization = `Bearer ${accessToken}`;
    }
    return config;
  },
  error => Promise.reject(error)
);

instance.interceptors.response.use(
  (response) => {
    const { data } = response;
    return (data && typeof data === 'object' && 'data' in data) ? data.data : data;
  },
  async (error: AxiosError) => {
    const originalRequest = error.config as RequestConfig;
    if (error.response?.status === 401 && !originalRequest?._retry && !originalRequest?.skipAuth) {
      originalRequest._retry = true;
      const { refreshToken } = getAuthTokens();
      if (refreshToken) {
        try {
          const res = await axios.post(`${env.NEXT_PUBLIC_API_URL}/auth/refresh`, { refreshToken });
          const { accessToken: nextAt, refreshToken: nextRt } = res.data.data || res.data;
          setAuthTokens(nextAt, nextRt);
          if (originalRequest.headers) originalRequest.headers.Authorization = `Bearer ${nextAt}`;
          return instance(originalRequest);
        } catch (refreshErr) {
          setAuthTokens(null, null);
          return Promise.reject(refreshErr);
        }
      }
    }

    const body = error.response?.data as any;
    const apiError = new Error(body?.message || body?.error || error.message);
    (apiError as any).status = error.response?.status;
    (apiError as any).code = body?.code;

    if (!originalRequest?.skipErrorHandler) {
      handleError(apiError);
    }
    return Promise.reject(apiError);
  }
);

// Purity: Wrap the instance to provide clean Promise<T> but avoid factory noise.
export const request = {
  get: <T = any>(url: string, config?: RequestConfig) => instance.get<any, T>(url, config),
  post: <T = any>(url: string, data?: any, config?: RequestConfig) => instance.post<any, T>(url, data, config),
  put: <T = any>(url: string, data?: any, config?: RequestConfig) => instance.put<any, T>(url, data, config),
  delete: <T = any>(url: string, config?: RequestConfig) => instance.delete<any, T>(url, config),
  patch: <T = any>(url: string, data?: any, config?: RequestConfig) => instance.patch<any, T>(url, data, config),
};

export class ApiError extends Error {
  code: string | number;
  status?: number;
  constructor(message: string, code: string | number, status?: number) {
    super(message);
    this.code = code;
    this.status = status;
  }
}

export default request;
