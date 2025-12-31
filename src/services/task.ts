import { z } from 'zod';
import { request } from '@/http';
import { components } from '@/types/api.generated';

// ============================================================================
// Types - Referenced from generated schema (Single Source of Truth)
// ============================================================================

export type Task = components['schemas']['Task'];
export type TaskCreateDto = components['schemas']['TaskCreateDto'];
export type TaskUpdateDto = components['schemas']['TaskUpdateDto'];
export type TaskListResponse = components['schemas']['TaskListResponse'];

// ============================================================================
// Zod Schemas - Runtime validation matching the contract
// ============================================================================

export const TaskSchema = z.object({
  id: z.string(),
  title: z.string().min(1),
  description: z.string().optional(),
  status: z.enum(['todo', 'doing', 'done']),
  priority: z.enum(['low', 'medium', 'high']),
  createdAt: z.string(),
});

export const TaskListResponseSchema = z.object({
  data: z.array(TaskSchema),
  total: z.number(),
});

// ============================================================================
// Task API
// ============================================================================

const ENDPOINTS = {
  BASE: '/backend/api/tasks',
  DETAIL: (id: string) => `/backend/api/tasks/${id}`,
} as const;

export const taskApi = {
  /**
   * List tasks with optional status filter
   */
  list: async (params?: { status?: Task['status'] }): Promise<TaskListResponse> => {
    const response = await request.get(ENDPOINTS.BASE, { params });
    return TaskListResponseSchema.parse(response);
  },

  /**
   * Get single task
   */
  get: async (id: string): Promise<Task> => {
    const response = await request.get(ENDPOINTS.DETAIL(id));
    return TaskSchema.parse(response);
  },

  /**
   * Create a new task
   */
  create: async (data: TaskCreateDto): Promise<Task> => {
    const response = await request.post<Task>(ENDPOINTS.BASE, data);
    return TaskSchema.parse(response);
  },

  /**
   * Update an existing task
   */
  update: async (id: string, data: TaskUpdateDto): Promise<Task> => {
    const response = await request.patch<Task>(ENDPOINTS.DETAIL(id), data);
    return TaskSchema.parse(response);
  },

  /**
   * Delete a task
   */
  delete: async (id: string): Promise<void> => {
    await request.delete(ENDPOINTS.DETAIL(id));
  },
} as const;
