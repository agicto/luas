/**
 * Example domain types
 * Following the project specification for type definitions.
 */

export interface ExampleItem {
  id: string;
  title: string;
  description?: string;
  status: 'active' | 'inactive';
  createdAt: string;
  updatedAt: string;
}

export interface CreateExampleRequest {
  title: string;
  description?: string;
  status?: 'active' | 'inactive';
}

export type UpdateExampleRequest = Partial<CreateExampleRequest>;

export interface ExampleQuerySchema {
  keyword?: string;
  status?: 'active' | 'inactive';
  page?: number;
  pageSize?: number;
}

export interface ExampleListResponse {
  items: ExampleItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface DeleteExampleResponse {
  id: string;
}
