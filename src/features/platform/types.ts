export interface GitHubConnection {
  id: number;
  name: string;
  provider: string;
  login: string;
  displayName: string;
  avatarUrl?: string;
  tokenMasked: string;
  scopes?: string[];
  lastSyncedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface GitHubRepository {
  id: number;
  name: string;
  fullName: string;
  owner: string;
  defaultBranch: string;
  cloneUrl: string;
  htmlUrl: string;
  private: boolean;
  description?: string;
  language?: string;
  updatedAt: string;
}

export interface Project {
  id: number;
  name: string;
  slug: string;
  description?: string;
  productionDomain?: string;
  serviceCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface ServiceEnvironmentVariable {
  id: number;
  key: string;
  value: string;
  isSecret: boolean;
}

export interface ServiceDeploymentSnapshot {
  id: string;
  status: 'pending' | 'running' | 'succeeded' | 'failed' | string;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
  lastLog?: string;
  error?: string;
}

export interface ManagedService {
  id: number;
  projectId: number;
  projectName: string;
  githubConnectionId: number;
  githubLogin: string;
  name: string;
  slug: string;
  repositoryOwner: string;
  repositoryName: string;
  repositoryUrl: string;
  defaultBranch: string;
  rootDirectory?: string;
  deployStrategy: 'dockerfile' | 'compose' | 'custom' | string;
  deployTarget: string;
  dockerfilePath?: string;
  composeFile?: string;
  buildCommand?: string;
  deployCommand?: string;
  healthCheckUrl?: string;
  domain?: string;
  publishedPort?: number;
  containerPort?: number;
  autoDeployEnabled: boolean;
  webhookSecret?: string;
  lastDeploymentId?: string;
  lastDeployment?: ServiceDeploymentSnapshot;
  lastDeployError?: string;
  environment: ServiceEnvironmentVariable[];
  createdAt: string;
  updatedAt: string;
}

export interface PlatformOverview {
  projects: number;
  services: number;
  githubConnections: number;
  recentDeployments: number;
  projectsList: Project[];
  servicesList: ManagedService[];
  deployments: ServiceDeploymentSnapshot[];
}

export interface DeployTarget {
  name: string;
  displayName: string;
  provider: string;
  workingDirectory: string;
  domain?: string;
  certificateMode?: string;
  environment?: Record<string, string>;
}

export interface DeploymentLogEntry {
  sequence: number;
  deploymentId: string;
  timestamp: string;
  stream: string;
  message: string;
}

export interface DeploymentWatchEvent {
  deployment?: {
    id: string;
    status: string;
    lastLog?: string;
    error?: string;
    createdAt: string;
    startedAt?: string;
    finishedAt?: string;
  };
  log?: DeploymentLogEntry;
  done: boolean;
}

export interface ConnectGitHubRequest {
  name?: string;
  token: string;
}

export interface ServiceEnvironmentVariableInput {
  key: string;
  value: string;
  isSecret: boolean;
}

export interface ImportServiceRequest {
  projectId?: number;
  projectName?: string;
  projectDescription?: string;
  projectProductionDomain?: string;
  githubConnectionId: number;
  name?: string;
  repositoryOwner: string;
  repositoryName: string;
  repositoryUrl?: string;
  branch: string;
  rootDirectory?: string;
  deployStrategy: 'dockerfile' | 'compose' | 'custom';
  deployTarget: string;
  dockerfilePath?: string;
  composeFile?: string;
  buildCommand?: string;
  deployCommand?: string;
  healthCheckUrl?: string;
  domain?: string;
  publishedPort?: number;
  containerPort?: number;
  autoDeployEnabled?: boolean;
  environment?: ServiceEnvironmentVariableInput[];
}

export interface ImportServiceResponse {
  service: ManagedService;
}

export interface ReplaceEnvironmentRequest {
  variables: ServiceEnvironmentVariableInput[];
}

export interface TriggerServiceDeploymentRequest {
  branch?: string;
  commit?: string;
  triggeredBy?: string;
}

export interface TriggerServiceDeploymentResponse {
  service: ManagedService;
  deployment: {
    id: string;
    status: string;
    lastLog?: string;
    error?: string;
    createdAt: string;
    startedAt?: string;
    finishedAt?: string;
  };
}
