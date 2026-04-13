import { createRequest } from '@/http';
import { env } from '@/config/env';
import type {
  CertificateInfo,
  DeployTarget,
  Deployment,
  DeploymentLogEntry,
  GenerateCertificateRequest,
  StartDeploymentRequest,
  StartDeploymentResponse,
} from '@/features/deploy/types';

const deployRequest = createRequest({
  baseURL: env.NEXT_PUBLIC_DEPLOY_API_URL,
});

function normalizeBaseUrl(value: string): string {
  return value.endsWith('/') ? value.slice(0, -1) : value;
}

export const deployService = {
  getTargets: () => deployRequest.get<DeployTarget[]>('/deploy/targets'),

  getDeployments: (limit = 20) =>
    deployRequest.get<Deployment[]>('/deployments', {
      params: { limit },
    }),

  getDeployment: (id: string) => deployRequest.get<Deployment>(`/deployments/${id}`),

  getLogs: (id: string, tail = 200) =>
    deployRequest.get<DeploymentLogEntry[]>(`/deployments/${id}/logs`, {
      params: { tail },
    }),

  startDeployment: (payload: StartDeploymentRequest) =>
    deployRequest.post<StartDeploymentResponse, StartDeploymentRequest>('/deployments', payload),

  generateCertificate: (payload: GenerateCertificateRequest) =>
    deployRequest.post<CertificateInfo, GenerateCertificateRequest>('/deployments/certificates', payload),

  streamUrl: (id: string) =>
    `${normalizeBaseUrl(env.NEXT_PUBLIC_DEPLOY_API_URL)}/deployments/${id}/stream`,
};

export type DeployService = typeof deployService;
