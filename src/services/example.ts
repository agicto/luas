/**
 * Example service layer
 * Stateless pure functions for API interactions.
 * 
 * @see src/http/request.ts for the underlying request implementation.
 */

import { request } from '@/http';
import type { 
  ExampleItem, 
  CreateExampleRequest, 
  UpdateExampleRequest, 
  ExampleQuerySchema 
} from '@/types/example';

const ENDPOINTS = {
  BASE: '/example-items',
  DETAIL: (id: string) => `/example-items/${id}`,
} as const;

export const exampleService = {
  /**
   * Fetch a list of example items with optional filtering
   */
  getList: (params?: ExampleQuerySchema) =>
    request.get<ExampleItem[]>(ENDPOINTS.BASE, { params }),

  /**
   * Fetch a single item by its ID
   */
  getDetail: (id: string) =>
    request.get<ExampleItem>(ENDPOINTS.DETAIL(id)),

  /**
   * Create a new item
   */
  create: (data: CreateExampleRequest) =>
    request.post<ExampleItem>(ENDPOINTS.BASE, data),

  /**
   * Update an existing item
   */
  update: (id: string, data: UpdateExampleRequest) =>
    request.put<ExampleItem>(ENDPOINTS.DETAIL(id), data),

  /**
   * Delete an item
   */
  delete: (id: string) =>
    request.delete(ENDPOINTS.DETAIL(id)),
};

export type ExampleService = typeof exampleService;
