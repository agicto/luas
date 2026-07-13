import axios, { AxiosError, AxiosInstance, AxiosRequestConfig } from 'axios';
import { env } from '@/config/env';
import {
  ClientErrorCode,
  HttpStatusErrorCodeMap,
  normalizeLegacyErrorCode,
  type ErrorCodeValue,
} from './codes';

/** Axios request options accepted by HttpClient. */
export type RequestConfig = AxiosRequestConfig;

interface ApiErrorBody {
  code?: string | number;
  error_code?: string;
  error?: string;
  errors?: ApiFieldErrors;
  message?: string;
  request_id?: string;
}

export type ApiFieldErrors = Record<string, string[]>;

/**
 * ApiError Class to encapsulate API-related errors
 */
export class ApiError extends Error {
  errorCode: ErrorCodeValue;
  fieldErrors?: ApiFieldErrors;
  status?: number;
  requestId?: string;

  constructor(
    message: string,
    errorCode: ErrorCodeValue,
    status?: number,
    requestId?: string,
    fieldErrors?: ApiFieldErrors
  ) {
    super(message);
    this.name = 'ApiError';
    this.errorCode = errorCode;
    this.status = status;
    this.requestId = requestId;
    this.fieldErrors = fieldErrors;
  }
}

function clientErrorCodeFor(error: AxiosError): ErrorCodeValue {
  const axiosCode = error.code?.toUpperCase();

  if (axiosCode === 'ECONNABORTED' || axiosCode === 'ETIMEDOUT') {
    return ClientErrorCode.TIMEOUT;
  }

  if (!error.response) {
    return ClientErrorCode.NETWORK_ERROR;
  }

  return ClientErrorCode.FETCH_ERROR;
}

export function toApiError(error: AxiosError): ApiError {
  const body = error.response?.data as ApiErrorBody | undefined;
  const status = error.response?.status;
  const legacyErrorCode = normalizeLegacyErrorCode(body?.code);

  return new ApiError(
    body?.message ?? body?.error ?? error.message,
    body?.error_code ??
      legacyErrorCode ??
      (status ? HttpStatusErrorCodeMap[status] : undefined) ??
      clientErrorCodeFor(error),
    status,
    body?.request_id,
    body?.errors
  );
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
    // Response interceptor: envelope extraction and error normalization.
    this.instance.interceptors.response.use(
      (response) => {
        const { data } = response;
        // Standard payload extraction for { code, data, message } responses.
        return data && typeof data === 'object' && 'data' in data ? data.data : data;
      },
      (error: AxiosError) => Promise.reject(toApiError(error))
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
