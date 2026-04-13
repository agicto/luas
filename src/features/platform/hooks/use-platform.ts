'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { platformService } from '@/features/platform/services/platform-service';
import type {
  ConnectGitHubRequest,
  ImportServiceRequest,
  ReplaceEnvironmentRequest,
  TriggerServiceDeploymentRequest,
} from '@/features/platform/types';

const platformKeys = {
  all: ['platform'] as const,
  overview: () => [...platformKeys.all, 'overview'] as const,
  targets: () => [...platformKeys.all, 'targets'] as const,
  connections: () => [...platformKeys.all, 'connections'] as const,
  repositories: (connectionId: number | null, query: string) =>
    [...platformKeys.all, 'repositories', connectionId ?? 'none', query] as const,
  projects: () => [...platformKeys.all, 'projects'] as const,
  services: () => [...platformKeys.all, 'services'] as const,
  service: (serviceId: number | null) => [...platformKeys.all, 'service', serviceId ?? 'none'] as const,
  deployments: (serviceId: number | null, limit: number) =>
    [...platformKeys.all, 'deployments', serviceId ?? 'none', limit] as const,
  logs: (deploymentId: string | null, tail: number) =>
    [...platformKeys.all, 'logs', deploymentId ?? 'none', tail] as const,
};

export function usePlatformOverview() {
  return useQuery({
    queryKey: platformKeys.overview(),
    queryFn: platformService.getOverview,
    refetchInterval: 15_000,
  });
}

export function useDeployTargets() {
  return useQuery({
    queryKey: platformKeys.targets(),
    queryFn: platformService.getDeployTargets,
  });
}

export function useGitHubConnections() {
  return useQuery({
    queryKey: platformKeys.connections(),
    queryFn: platformService.getGitHubConnections,
  });
}

export function useGitHubRepositories(connectionId: number | null, query: string) {
  return useQuery({
    queryKey: platformKeys.repositories(connectionId, query),
    queryFn: () => platformService.getGitHubRepositories(connectionId ?? 0, query),
    enabled: Boolean(connectionId),
    staleTime: 20_000,
  });
}

export function useProjects() {
  return useQuery({
    queryKey: platformKeys.projects(),
    queryFn: platformService.getProjects,
  });
}

export function useServices() {
  return useQuery({
    queryKey: platformKeys.services(),
    queryFn: platformService.getServices,
    refetchInterval: 12_000,
  });
}

export function useService(serviceId: number | null) {
  return useQuery({
    queryKey: platformKeys.service(serviceId),
    queryFn: () => platformService.getService(serviceId ?? 0),
    enabled: Boolean(serviceId),
    refetchInterval: 8_000,
  });
}

export function useServiceDeployments(serviceId: number | null, limit = 20) {
  return useQuery({
    queryKey: platformKeys.deployments(serviceId, limit),
    queryFn: () => platformService.getServiceDeployments(serviceId ?? 0, limit),
    enabled: Boolean(serviceId),
    refetchInterval: 8_000,
  });
}

export function useDeploymentLogs(deploymentId: string | null, tail = 300) {
  return useQuery({
    queryKey: platformKeys.logs(deploymentId, tail),
    queryFn: () => platformService.getDeploymentLogs(deploymentId ?? '', tail),
    enabled: Boolean(deploymentId),
  });
}

export function useConnectGitHub() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: ConnectGitHubRequest) => platformService.connectGitHub(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: platformKeys.all });
    },
  });
}

export function useImportService() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: ImportServiceRequest) => platformService.importService(payload),
    onSuccess: ({ service }) => {
      queryClient.invalidateQueries({ queryKey: platformKeys.all });
      queryClient.setQueryData(platformKeys.service(service.id), service);
    },
  });
}

export function useReplaceEnvironment(serviceId: number | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: ReplaceEnvironmentRequest) =>
      platformService.replaceEnvironment(serviceId ?? 0, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: platformKeys.all });
    },
  });
}

export function useDeployService(serviceId: number | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: TriggerServiceDeploymentRequest) =>
      platformService.deployService(serviceId ?? 0, payload),
    onSuccess: ({ service }) => {
      queryClient.invalidateQueries({ queryKey: platformKeys.all });
      queryClient.setQueryData(platformKeys.service(service.id), service);
    },
  });
}
