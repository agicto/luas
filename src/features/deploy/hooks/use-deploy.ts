'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { deployService } from '@/features/deploy/services/deploy-service';
import type {
  GenerateCertificateRequest,
  StartDeploymentRequest,
} from '@/features/deploy/types';

const deployKeys = {
  all: ['deploy'] as const,
  targets: () => [...deployKeys.all, 'targets'] as const,
  deployments: (limit: number) => [...deployKeys.all, 'deployments', limit] as const,
  deployment: (id: string) => [...deployKeys.all, 'deployment', id] as const,
  logs: (id: string, tail: number) => [...deployKeys.all, 'logs', id, tail] as const,
};

export function useDeployTargets() {
  return useQuery({
    queryKey: deployKeys.targets(),
    queryFn: deployService.getTargets,
  });
}

export function useDeployments(limit = 20) {
  return useQuery({
    queryKey: deployKeys.deployments(limit),
    queryFn: () => deployService.getDeployments(limit),
    refetchInterval: 10_000,
  });
}

export function useDeployment(id: string | null) {
  return useQuery({
    queryKey: deployKeys.deployment(id ?? 'none'),
    queryFn: () => deployService.getDeployment(id ?? ''),
    enabled: Boolean(id),
    refetchInterval: (query) => {
      const deployment = query.state.data;
      return deployment?.status === 'running' || deployment?.status === 'pending' ? 5_000 : false;
    },
  });
}

export function useDeploymentLogs(id: string | null, tail = 200) {
  return useQuery({
    queryKey: deployKeys.logs(id ?? 'none', tail),
    queryFn: () => deployService.getLogs(id ?? '', tail),
    enabled: Boolean(id),
  });
}

export function useStartDeployment() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: StartDeploymentRequest) => deployService.startDeployment(payload),
    onSuccess: ({ deployment }) => {
      queryClient.invalidateQueries({ queryKey: deployKeys.all });
      queryClient.setQueryData(deployKeys.deployment(deployment.id), deployment);
    },
  });
}

export function useGenerateCertificate() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: GenerateCertificateRequest) => deployService.generateCertificate(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: deployKeys.all });
    },
  });
}
