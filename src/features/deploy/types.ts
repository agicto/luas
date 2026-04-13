export type DeploymentStatus = 'pending' | 'running' | 'succeeded' | 'failed';

export type CertificateMode = 'disabled' | 'self-signed' | 'render-managed';

export interface DeployTarget {
  name: string;
  displayName: string;
  provider: string;
  workingDirectory: string;
  buildCommand?: string;
  deployCommand: string;
  healthCheckUrl?: string;
  healthCheckTimeout?: string;
  environment?: Record<string, string>;
  domain?: string;
  certificateMode?: CertificateMode;
  autoDeployBranches?: string[];
}

export interface CertificateInfo {
  domain: string;
  certPath: string;
  keyPath: string;
  generatedAt: string;
  expiresAt: string;
  mode: string;
}

export interface Deployment {
  id: string;
  target: string;
  targetName: string;
  provider: string;
  status: DeploymentStatus;
  triggeredBy: string;
  triggerMode: string;
  branch?: string;
  commit?: string;
  workingDirectory: string;
  command: string;
  environment?: Record<string, string>;
  domain?: string;
  healthCheckUrl?: string;
  certificateMode: CertificateMode;
  certificate?: CertificateInfo;
  lastLog?: string;
  error?: string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
}

export interface DeploymentLogEntry {
  sequence: number;
  deploymentId: string;
  timestamp: string;
  stream: string;
  message: string;
}

export interface DeploymentWatchEvent {
  deployment?: Deployment;
  log?: DeploymentLogEntry;
  done: boolean;
}

export interface StartDeploymentRequest {
  target: string;
  branch?: string;
  commit?: string;
  triggeredBy?: string;
  environment?: Record<string, string>;
}

export interface StartDeploymentResponse {
  deployment: Deployment;
}

export interface GenerateCertificateRequest {
  domain: string;
  validDays?: number;
}
