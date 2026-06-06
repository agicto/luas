export interface AccessPermission {
  key: string;
  label: string;
  category: string;
}

export interface AccessRole {
  id: number;
  team_id: number;
  name: string;
  slug: string;
  description?: string;
  permissions: string[];
  system: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateAccessRoleRequest {
  name: string;
  slug?: string;
  description?: string;
  permissions: string[];
}

export interface UpdateAccessRoleRequest {
  name?: string;
  description?: string;
  permissions?: string[];
}

export interface PaginatedAccessRoleResponse {
  data: AccessRole[];
  meta?: unknown;
  links?: unknown;
}
