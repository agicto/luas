import axios, { AxiosError, AxiosInstance, AxiosRequestConfig } from 'axios';
import { env } from '@/config/env';
import {
  ClientErrorCode,
  HttpStatusErrorCodeMap,
  normalizeLegacyErrorCode,
  type ErrorCodeValue,
} from './codes';

type ResponseMode = 'data' | 'envelope';

/** Axios request options accepted by HttpClient. */
export type RequestConfig = AxiosRequestConfig & {
  luasResponseMode?: ResponseMode;
};

export interface ApiSuccessEnvelope<T = unknown> {
  code: 0;
  message: string;
  data: T;
  meta?: unknown;
  links?: unknown;
}

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
        const config = response.config as RequestConfig;
        if (config.luasResponseMode === 'envelope') {
          return data;
        }
        // Standard payload extraction for { code, data, message } responses.
        return data && typeof data === 'object' && 'data' in data ? data.data : data;
      },
      (error: AxiosError) => Promise.reject(toApiError(error))
    );
  }

  // The response interceptor resolves with the unwrapped payload instead of an AxiosResponse; axios
  // cannot express that for a generic payload type, so the result is narrowed here once.
  private payload<T>(response: Promise<unknown>): Promise<T> {
    return response as Promise<T>;
  }

  // Pure promise-based methods
  public get<T = unknown>(url: string, config?: RequestConfig): Promise<T> {
    return this.payload<T>(this.instance.get(url, config));
  }

  public getEnvelope<T = unknown>(
    url: string,
    config: RequestConfig = {}
  ): Promise<ApiSuccessEnvelope<T>> {
    const requestConfig: RequestConfig = {
      ...config,
      luasResponseMode: 'envelope',
    };
    return this.instance.get<ApiSuccessEnvelope<T>, ApiSuccessEnvelope<T>>(
      url,
      requestConfig
    );
  }

  public post<T = unknown, D = unknown>(url: string, data?: D, config?: RequestConfig): Promise<T> {
    return this.payload<T>(this.instance.post<unknown, unknown, D>(url, data, config));
  }

  public put<T = unknown, D = unknown>(url: string, data?: D, config?: RequestConfig): Promise<T> {
    return this.payload<T>(this.instance.put<unknown, unknown, D>(url, data, config));
  }

  public patch<T = unknown, D = unknown>(url: string, data?: D, config?: RequestConfig): Promise<T> {
    return this.payload<T>(this.instance.patch<unknown, unknown, D>(url, data, config));
  }

  public delete<T = unknown>(url: string, config?: RequestConfig): Promise<T> {
    return this.payload<T>(this.instance.delete(url, config));
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
