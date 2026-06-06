import request from '@/http';
import type {
  CreateTeamRequest,
  PaginatedTeamResponse,
  Team,
  UpdateTeamRequest,
} from '@/features/team/types';

const basePath = '/teams';

export const teamService = {
  list: (params?: { page?: number; page_size?: number }) =>
    request.get<PaginatedTeamResponse>(basePath, { params }),

  get: (id: number) => request.get<Team>(`${basePath}/${id}`),

  create: (data: CreateTeamRequest) =>
    request.post<Team, CreateTeamRequest>(basePath, data),

  update: (id: number, data: UpdateTeamRequest) =>
    request.put<Team, UpdateTeamRequest>(`${basePath}/${id}`, data),

  delete: (id: number) => request.delete<void>(`${basePath}/${id}`),
};

export type TeamService = typeof teamService;
