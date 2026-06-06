export interface Team {
  id: number;
  name: string;
  slug: string;
  owner_user_id: number;
  plan: 'free' | string;
  status: 'active' | 'archived';
  created_at: string;
  updated_at: string;
}

export interface CreateTeamRequest {
  name: string;
  slug?: string;
}

export interface UpdateTeamRequest {
  name?: string;
  status?: Team['status'];
}

export interface PaginatedTeamResponse {
  data: Team[];
  meta?: unknown;
  links?: unknown;
}
