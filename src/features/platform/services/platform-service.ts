import { createRequest } from '@/http';
import { env } from '@/config/env';
import type {
  ConnectGitHubRequest,
  DeployTarget,
  DeploymentLogEntry,
  GitHubConnection,
  GitHubRepository,
  ImportServiceRequest,
  ImportServiceResponse,
  ManagedService,
  PlatformOverview,
  Project,
  ReplaceEnvironmentRequest,
  ServiceDeploymentSnapshot,
  ServiceEnvironmentVariable,
  TriggerServiceDeploymentRequest,
  TriggerServiceDeploymentResponse,
} from '@/features/platform/types';

const platformRequest = createRequest({
  baseURL: env.NEXT_PUBLIC_PLATFORM_API_URL,
});

function normalizeBaseUrl(value: string): string {
  return value.endsWith('/') ? value.slice(0, -1) : value;
}

export const platformService = {
  getOverview: () => platformRequest.get<PlatformOverview>('/platform/overview'),
  getDeployTargets: () => platformRequest.get<DeployTarget[]>('/platform/deploy-targets'),
  getGitHubConnections: () => platformRequest.get<GitHubConnection[]>('/platform/github/connections'),
  connectGitHub: (payload: ConnectGitHubRequest) =>
    platformRequest.post<GitHubConnection, ConnectGitHubRequest>('/platform/github/connections', payload),
  getGitHubRepositories: (connectionId: number, query = '', limit = 50) =>
    platformRequest.get<GitHubRepository[]>(`/platform/github/connections/${connectionId}/repositories`, {
      params: { query, limit },
    }),
  getProjects: () => platformRequest.get<Project[]>('/platform/projects'),
  getServices: () => platformRequest.get<ManagedService[]>('/platform/services'),
  getService: (id: number) => platformRequest.get<ManagedService>(`/platform/services/${id}`),
  importService: (payload: ImportServiceRequest) =>
    platformRequest.post<ImportServiceResponse, ImportServiceRequest>('/platform/services/import', payload),
  replaceEnvironment: (serviceId: number, payload: ReplaceEnvironmentRequest) =>
    platformRequest.put<ServiceEnvironmentVariable[], ReplaceEnvironmentRequest>(
      `/platform/services/${serviceId}/environment`,
      payload
    ),
  getServiceDeployments: (serviceId: number, limit = 20) =>
    platformRequest.get<ServiceDeploymentSnapshot[]>(`/platform/services/${serviceId}/deployments`, {
      params: { limit },
    }),
  deployService: (serviceId: number, payload: TriggerServiceDeploymentRequest) =>
    platformRequest.post<TriggerServiceDeploymentResponse, TriggerServiceDeploymentRequest>(
      `/platform/services/${serviceId}/deploy`,
      payload
    ),
  getDeploymentLogs: (deploymentId: string, tail = 300) =>
    platformRequest.get<DeploymentLogEntry[]>(`/platform/deployments/${deploymentId}/logs`, {
      params: { tail },
    }),
  streamUrl: (deploymentId: string) =>
    `${normalizeBaseUrl(env.NEXT_PUBLIC_PLATFORM_API_URL)}/platform/deployments/${deploymentId}/stream`,
};

export type PlatformService = typeof platformService;
