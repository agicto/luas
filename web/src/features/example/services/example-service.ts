import request from '@/http';
import type {
  CreateExampleRequest,
  DeleteExampleResponse,
  ExampleItem,
  ExampleListResponse,
  ExampleQuerySchema,
  UpdateExampleRequest,
} from '@/features/example/types';

/**
 * Example Service (Template)
 * 
 * Demonstrates the standard pattern for API service definitions.
 * All methods are stateless pure functions using the centralized request utility.
 */
export const exampleService = {
  /**
   * Fetch a paginated list of items
   */
  getList: (params?: ExampleQuerySchema) => 
    request.get<ExampleListResponse>('/example', { params }),
    
  /**
   * Fetch a single item by ID
   */
  getDetail: (id: string) => 
    request.get<ExampleItem>(`/example/${id}`),
    
  /**
   * Create a new item
   */
  create: (data: CreateExampleRequest) => 
    request.post<ExampleItem, CreateExampleRequest>('/example', data),
    
  /**
   * Update an existing item (Partial update)
   */
  update: (id: string, data: UpdateExampleRequest) => 
    request.patch<ExampleItem, UpdateExampleRequest>(`/example/${id}`, data),
    
  /**
   * Delete an item
   */
  delete: (id: string) => 
    request.delete<DeleteExampleResponse>(`/example/${id}`),
};

export type ExampleService = typeof exampleService;
