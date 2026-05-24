'use client';

import * as React from 'react';
import {
  Activity,
  Bot,
  Boxes,
  Cable,
  ChevronRight,
  ExternalLink,
  GitBranch,
  Github,
  Globe,
  PackageCheck,
  Rocket,
  Search,
  Server,
  TerminalSquare,
} from 'lucide-react';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Textarea } from '@/components/ui/textarea';
import { cn } from '@/utils';
import {
  useConnectGitHub,
  useDeployService,
  useDeployTargets,
  useDeploymentLogs,
  useGitHubConnections,
  useGitHubRepositories,
  useImportService,
  usePlatformOverview,
  useReplaceEnvironment,
  useService,
  useServiceDeployments,
  useServices,
} from '@/features/platform/hooks/use-platform';
import { platformService } from '@/features/platform/services/platform-service';
import type {
  DeploymentLogEntry,
  DeploymentWatchEvent,
  GitHubRepository,
} from '@/features/platform/types';

function statusTone(status?: string) {
  switch (status) {
    case 'succeeded':
      return 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300';
    case 'failed':
      return 'border-rose-500/25 bg-rose-500/10 text-rose-700 dark:text-rose-300';
    case 'running':
      return 'border-sky-500/25 bg-sky-500/10 text-sky-700 dark:text-sky-300';
    default:
      return 'border-amber-500/25 bg-amber-500/10 text-amber-700 dark:text-amber-300';
  }
}

function strategyTone(strategy: string) {
  switch (strategy) {
    case 'dockerfile':
      return 'bg-cyan-500/10 text-cyan-700 dark:text-cyan-300';
    case 'compose':
      return 'bg-orange-500/10 text-orange-700 dark:text-orange-300';
    default:
      return 'bg-slate-500/10 text-slate-700 dark:text-slate-300';
  }
}

function formatTime(value?: string) {
  if (!value) {
    return 'Pending';
  }

  return new Intl.DateTimeFormat('en-IE', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value));
}

function parseEnvironmentInput(source: string) {
  const rows = source
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);

  return rows.map((entry) => {
    const index = entry.indexOf('=');
    if (index <= 0) {
      throw new Error(`Invalid env line: ${entry}`);
    }

    return {
      key: entry.slice(0, index).trim(),
      value: entry.slice(index + 1).trim(),
      isSecret: true,
    };
  });
}

function stringifyEnvironment(
  variables: { key: string; value: string }[] | undefined
) {
  if (!variables || variables.length === 0) {
    return '';
  }

  return variables.map((item) => `${item.key}=${item.value}`).join('\n');
}

export default function ConsolePage() {
  const { data: overview } = usePlatformOverview();
  const { data: targets = [] } = useDeployTargets();
  const { data: connections = [] } = useGitHubConnections();
  const { data: services = [], refetch: refetchServices } = useServices();

  const [connectionId, setConnectionId] = React.useState<number | null>(null);
  const [repoQuery, setRepoQuery] = React.useState('');
  const deferredRepoQuery = React.useDeferredValue(repoQuery);
  const { data: repositories = [] } = useGitHubRepositories(connectionId, deferredRepoQuery);

  const [selectedRepository, setSelectedRepository] = React.useState<GitHubRepository | null>(null);
  const [selectedServiceId, setSelectedServiceId] = React.useState<number | null>(null);
  const { data: selectedService } = useService(selectedServiceId);
  const { data: deployments = [], refetch: refetchDeployments } = useServiceDeployments(
    selectedServiceId,
    20
  );

  const [selectedDeploymentId, setSelectedDeploymentId] = React.useState<string | null>(null);
  const { data: deploymentLogs = [] } = useDeploymentLogs(selectedDeploymentId, 400);

  const connectGitHub = useConnectGitHub();
  const importService = useImportService();
  const replaceEnvironment = useReplaceEnvironment(selectedServiceId);
  const deployService = useDeployService(selectedServiceId);

  const [connectionAlias, setConnectionAlias] = React.useState('');
  const [connectionToken, setConnectionToken] = React.useState('');

  const [projectName, setProjectName] = React.useState('');
  const [projectDomain, setProjectDomain] = React.useState('');
  const [serviceName, setServiceName] = React.useState('');
  const [branch, setBranch] = React.useState('main');
  const [targetName, setTargetName] = React.useState('');
  const [strategy, setStrategy] = React.useState<'dockerfile' | 'compose' | 'custom'>('dockerfile');
  const [rootDirectory, setRootDirectory] = React.useState('');
  const [dockerfilePath, setDockerfilePath] = React.useState('Dockerfile');
  const [composeFile, setComposeFile] = React.useState('docker-compose.yml');
  const [buildCommand, setBuildCommand] = React.useState('');
  const [deployCommand, setDeployCommand] = React.useState('');
  const [healthCheckUrl, setHealthCheckUrl] = React.useState('');
  const [domain, setDomain] = React.useState('');
  const [publishedPort, setPublishedPort] = React.useState('3000');
  const [containerPort, setContainerPort] = React.useState('3000');
  const [autoDeployEnabled, setAutoDeployEnabled] = React.useState(true);
  const [importEnv, setImportEnv] = React.useState('');

  const [environmentEditor, setEnvironmentEditor] = React.useState('');
  const [streamedLogs, setStreamedLogs] = React.useState<DeploymentLogEntry[]>([]);
  const deferredLogs = React.useDeferredValue(streamedLogs);
  const logViewportRef = React.useRef<HTMLDivElement | null>(null);

  React.useEffect(() => {
    if (connectionId === null && connections.length > 0) {
      setConnectionId(connections[0].id);
    }
  }, [connectionId, connections]);

  React.useEffect(() => {
    if (!targetName && targets.length > 0) {
      setTargetName(targets[0].name);
    }
  }, [targetName, targets]);

  React.useEffect(() => {
    if (!selectedServiceId && services.length > 0) {
      setSelectedServiceId(services[0].id);
    }
  }, [selectedServiceId, services]);

  React.useEffect(() => {
    if (!selectedDeploymentId && deployments.length > 0) {
      setSelectedDeploymentId(deployments[0].id);
    }
  }, [deployments, selectedDeploymentId]);

  React.useEffect(() => {
    if (!selectedRepository) {
      return;
    }

    setProjectName((current) => current || selectedRepository.owner);
    setServiceName((current) => current || selectedRepository.name);
    setBranch(selectedRepository.defaultBranch);
  }, [selectedRepository]);

  React.useEffect(() => {
    setEnvironmentEditor(stringifyEnvironment(selectedService?.environment));
  }, [selectedService]);

  React.useEffect(() => {
    setStreamedLogs(deploymentLogs);
  }, [deploymentLogs, selectedDeploymentId]);

  React.useEffect(() => {
    if (!selectedDeploymentId) {
      return;
    }

    const eventSource = new EventSource(platformService.streamUrl(selectedDeploymentId), {
      withCredentials: true,
    });

    eventSource.onmessage = (message) => {
      try {
        const payload = JSON.parse(message.data) as DeploymentWatchEvent;

        if (payload.log) {
          const logEntry = payload.log;
          React.startTransition(() => {
            setStreamedLogs((current) => {
              if (current.some((entry) => entry.sequence === logEntry.sequence)) {
                return current;
              }

              const next = [...current, logEntry];
              return next.slice(-500);
            });
          });
        }

        if (payload.done) {
          void refetchServices();
          void refetchDeployments();
          eventSource.close();
        }
      } catch {
        eventSource.close();
      }
    };

    eventSource.onerror = () => {
      eventSource.close();
    };

    return () => {
      eventSource.close();
    };
  }, [refetchDeployments, refetchServices, selectedDeploymentId]);

  React.useEffect(() => {
    if (!logViewportRef.current) {
      return;
    }
    logViewportRef.current.scrollTop = logViewportRef.current.scrollHeight;
  }, [deferredLogs]);

  async function handleConnectGitHub() {
    if (!connectionToken.trim()) {
      toast.error('Paste a GitHub personal access token first.');
      return;
    }

    const connection = await connectGitHub.mutateAsync({
      name: connectionAlias.trim() || undefined,
      token: connectionToken.trim(),
    });

    setConnectionId(connection.id);
    setConnectionToken('');
    setConnectionAlias('');
    toast.success(`Connected ${connection.login}`);
  }

  async function handleImportService() {
    if (!selectedRepository) {
      toast.error('Choose a repository to import.');
      return;
    }
    if (!connectionId) {
      toast.error('Choose a GitHub connection first.');
      return;
    }
    if (!targetName) {
      toast.error('Choose a deploy target first.');
      return;
    }

    let environment;
    try {
      environment = importEnv.trim() ? parseEnvironmentInput(importEnv) : undefined;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Invalid environment variables.');
      return;
    }

    const result = await importService.mutateAsync({
      githubConnectionId: connectionId,
      projectName: projectName.trim() || selectedRepository.owner,
      projectProductionDomain: projectDomain.trim() || undefined,
      name: serviceName.trim() || selectedRepository.name,
      repositoryOwner: selectedRepository.owner,
      repositoryName: selectedRepository.name,
      repositoryUrl: selectedRepository.cloneUrl,
      branch: branch.trim() || selectedRepository.defaultBranch,
      rootDirectory: rootDirectory.trim() || undefined,
      deployStrategy: strategy,
      deployTarget: targetName,
      dockerfilePath: strategy === 'dockerfile' ? dockerfilePath.trim() || undefined : undefined,
      composeFile: strategy === 'compose' ? composeFile.trim() || undefined : undefined,
      buildCommand: strategy === 'custom' ? buildCommand.trim() || undefined : undefined,
      deployCommand: strategy === 'custom' ? deployCommand.trim() || undefined : undefined,
      healthCheckUrl: healthCheckUrl.trim() || undefined,
      domain: domain.trim() || undefined,
      publishedPort: strategy === 'dockerfile' ? Number.parseInt(publishedPort, 10) : undefined,
      containerPort: strategy === 'dockerfile' ? Number.parseInt(containerPort, 10) : undefined,
      autoDeployEnabled,
      environment,
    });

    setSelectedServiceId(result.service.id);
    toast.success(`Imported ${result.service.name}`);
  }

  async function handleSaveEnvironment() {
    if (!selectedServiceId) {
      return;
    }

    let variables;
    try {
      variables = environmentEditor.trim() ? parseEnvironmentInput(environmentEditor) : [];
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Invalid environment variables.');
      return;
    }

    await replaceEnvironment.mutateAsync({ variables });
    toast.success('Environment saved');
  }

  async function handleDeploySelectedService() {
    if (!selectedServiceId) {
      toast.error('Choose a service first.');
      return;
    }

    const result = await deployService.mutateAsync({
      triggeredBy: 'console',
    });

    setSelectedDeploymentId(result.deployment.id);
    setStreamedLogs([]);
    toast.success(`Deployment ${result.deployment.id.slice(0, 8)} started`);
  }

  const selectedTarget = targets.find((target) => target.name === targetName);
  const selectedServiceTarget = targets.find((target) => target.name === selectedService?.deployTarget);

  return (
    <div className="flex-1 space-y-6 p-6 pt-6">
      <section className="relative overflow-hidden rounded-[32px] border border-border/50 bg-[radial-gradient(circle_at_top_left,_rgba(16,185,129,0.18),_transparent_35%),radial-gradient(circle_at_bottom_right,_rgba(14,165,233,0.2),_transparent_30%),linear-gradient(135deg,_rgba(2,6,23,1),_rgba(15,23,42,0.98))] p-6 text-white shadow-2xl">
        <div className="absolute inset-0 opacity-30 [background-image:linear-gradient(rgba(255,255,255,0.08)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.08)_1px,transparent_1px)] [background-size:30px_30px]" />
        <div className="relative flex flex-col gap-6 xl:flex-row xl:items-end xl:justify-between">
          <div className="max-w-3xl space-y-3">
            <Badge className="rounded-full border border-white/15 bg-white/10 px-3 py-1 text-white">
              Luas Platform Control Plane
            </Badge>
            <div className="space-y-2">
              <h1 className="text-3xl font-semibold tracking-tight md:text-4xl">
                从 GitHub 仓库直接部署，不再手写发布链路
              </h1>
              <p className="max-w-2xl text-sm leading-6 text-white/72 md:text-base">
                连接 GitHub，选择仓库，定义部署策略和目标节点，平台会生成服务模型、保存环境变量、触发部署并实时回传日志。
              </p>
            </div>
          </div>

          <div className="grid gap-3 sm:grid-cols-4">
            {[
              { label: 'Projects', value: overview?.projects ?? 0, icon: Boxes },
              { label: 'Services', value: overview?.services ?? 0, icon: Server },
              { label: 'GitHub', value: overview?.githubConnections ?? 0, icon: Github },
              { label: 'Deployments', value: overview?.recentDeployments ?? 0, icon: Activity },
            ].map((item) => (
              <div
                key={item.label}
                className="rounded-2xl border border-white/10 bg-white/8 px-4 py-3 backdrop-blur"
              >
                <div className="flex items-center justify-between text-xs uppercase tracking-[0.24em] text-white/50">
                  <span>{item.label}</span>
                  <item.icon className="h-4 w-4" />
                </div>
                <div className="mt-3 text-2xl font-semibold">{item.value}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="grid gap-6 xl:grid-cols-[420px_minmax(0,1fr)]">
        <Card className="border-border/60 bg-card/95 shadow-lg">
          <CardHeader className="border-b border-border/50 bg-muted/25">
            <CardTitle className="flex items-center gap-2">
              <Cable className="h-5 w-5 text-primary" />
              GitHub 连接
            </CardTitle>
            <CardDescription>录入 Token 后立刻同步账号和可访问仓库。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="github-alias">连接名称</Label>
              <Input
                id="github-alias"
                value={connectionAlias}
                onChange={(event) => setConnectionAlias(event.target.value)}
                placeholder="例如：个人账号 / 团队机器人"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="github-token">GitHub Token</Label>
              <Textarea
                id="github-token"
                value={connectionToken}
                onChange={(event) => setConnectionToken(event.target.value)}
                placeholder="ghp_xxx / github_pat_xxx"
                className="min-h-28 font-mono text-xs"
              />
            </div>

            <Button
              className="w-full"
              loading={connectGitHub.isPending}
              onClick={handleConnectGitHub}
            >
              <Github className="h-4 w-4" />
              连接 GitHub
            </Button>

            <div className="space-y-3">
              <div className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                已连接账号
              </div>
              <div className="space-y-2">
                {connections.map((connection) => {
                  const active = connection.id === connectionId;
                  return (
                    <button
                      key={connection.id}
                      type="button"
                      onClick={() => setConnectionId(connection.id)}
                      className={cn(
                        'flex w-full items-center justify-between rounded-2xl border px-4 py-3 text-left transition',
                        active
                          ? 'border-primary/45 bg-primary/8 shadow-sm'
                          : 'border-border/60 bg-muted/15 hover:border-primary/25'
                      )}
                    >
                      <div className="space-y-1">
                        <div className="font-medium">{connection.displayName}</div>
                        <div className="text-xs text-muted-foreground">
                          @{connection.login} · {connection.tokenMasked}
                        </div>
                      </div>
                      <ChevronRight className="h-4 w-4 text-muted-foreground" />
                    </button>
                  );
                })}
                {connections.length === 0 ? (
                  <div className="rounded-2xl border border-dashed border-border/70 bg-muted/15 p-4 text-sm text-muted-foreground">
                    还没有 GitHub 连接。先接一个账号，右侧仓库列表才会出现。
                  </div>
                ) : null}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/60 bg-card/95 shadow-lg">
          <CardHeader className="border-b border-border/50 bg-muted/20">
            <CardTitle className="flex items-center gap-2">
              <Github className="h-5 w-5 text-primary" />
              仓库导入工作台
            </CardTitle>
            <CardDescription>选仓库后直接创建服务，平台按策略生成部署流程。</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_380px]">
            <div className="space-y-4">
              <div className="flex items-center gap-3">
                <div className="relative flex-1">
                  <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    value={repoQuery}
                    onChange={(event) => setRepoQuery(event.target.value)}
                    placeholder="搜索仓库名、语言、描述"
                    className="pl-9"
                  />
                </div>
                <Badge variant="outline" className="rounded-full px-3 py-1">
                  {repositories.length} repos
                </Badge>
              </div>

              <div className="max-h-[520px] overflow-auto rounded-2xl border border-border/60">
                <Table>
                  <TableHeader className="bg-muted/20">
                    <TableRow>
                      <TableHead>Repository</TableHead>
                      <TableHead>Branch</TableHead>
                      <TableHead>Updated</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {repositories.map((repository) => {
                      const active = selectedRepository?.id === repository.id;
                      return (
                        <TableRow
                          key={repository.id}
                          className={cn('cursor-pointer hover:bg-muted/25', active && 'bg-primary/6')}
                          onClick={() => setSelectedRepository(repository)}
                        >
                          <TableCell>
                            <div className="space-y-1">
                              <div className="font-medium">{repository.fullName}</div>
                              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                                <span>{repository.language || 'Unknown'}</span>
                                <span>{repository.private ? 'Private' : 'Public'}</span>
                              </div>
                              {repository.description ? (
                                <div className="line-clamp-2 text-xs text-muted-foreground">
                                  {repository.description}
                                </div>
                              ) : null}
                            </div>
                          </TableCell>
                          <TableCell className="font-mono text-xs">{repository.defaultBranch}</TableCell>
                          <TableCell className="text-xs text-muted-foreground">
                            {formatTime(repository.updatedAt)}
                          </TableCell>
                        </TableRow>
                      );
                    })}
                    {repositories.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={3} className="py-8 text-center text-sm text-muted-foreground">
                          选一个 GitHub 连接后，这里会列出仓库。
                        </TableCell>
                      </TableRow>
                    ) : null}
                  </TableBody>
                </Table>
              </div>
            </div>

            <div className="space-y-4 rounded-[28px] border border-border/60 bg-[linear-gradient(180deg,rgba(15,23,42,0.02),rgba(15,23,42,0.08))] p-5">
              <div className="space-y-1">
                <div className="text-lg font-semibold">创建服务</div>
                <div className="text-sm text-muted-foreground">
                  {selectedRepository ? selectedRepository.fullName : '先从左侧点一个仓库'}
                </div>
              </div>

              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-1">
                <div className="space-y-2">
                  <Label>项目名</Label>
                  <Input value={projectName} onChange={(event) => setProjectName(event.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label>服务名</Label>
                  <Input value={serviceName} onChange={(event) => setServiceName(event.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label>分支</Label>
                  <Input value={branch} onChange={(event) => setBranch(event.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label>部署目标</Label>
                  <Select value={targetName} onValueChange={setTargetName}>
                    <SelectTrigger>
                      <SelectValue placeholder="选择目标节点" />
                    </SelectTrigger>
                    <SelectContent>
                      {targets.map((target) => (
                        <SelectItem key={target.name} value={target.name}>
                          {target.displayName}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="space-y-2">
                <Label>部署策略</Label>
                <Select value={strategy} onValueChange={(value) => setStrategy(value as typeof strategy)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="dockerfile">Dockerfile</SelectItem>
                    <SelectItem value="compose">Docker Compose</SelectItem>
                    <SelectItem value="custom">Custom Commands</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label>根目录</Label>
                <Input
                  value={rootDirectory}
                  onChange={(event) => setRootDirectory(event.target.value)}
                  placeholder="apps/web"
                />
              </div>

              {strategy === 'dockerfile' ? (
                <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-1">
                  <div className="space-y-2">
                    <Label>Dockerfile 路径</Label>
                    <Input
                      value={dockerfilePath}
                      onChange={(event) => setDockerfilePath(event.target.value)}
                    />
                  </div>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-2">
                      <Label>外部端口</Label>
                      <Input
                        value={publishedPort}
                        onChange={(event) => setPublishedPort(event.target.value)}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>容器端口</Label>
                      <Input
                        value={containerPort}
                        onChange={(event) => setContainerPort(event.target.value)}
                      />
                    </div>
                  </div>
                </div>
              ) : null}

              {strategy === 'compose' ? (
                <div className="space-y-2">
                  <Label>Compose 文件</Label>
                  <Input value={composeFile} onChange={(event) => setComposeFile(event.target.value)} />
                </div>
              ) : null}

              {strategy === 'custom' ? (
                <div className="space-y-4">
                  <div className="space-y-2">
                    <Label>Build Command</Label>
                    <Textarea
                      value={buildCommand}
                      onChange={(event) => setBuildCommand(event.target.value)}
                      placeholder="pnpm install && pnpm build"
                      className="min-h-24 font-mono text-xs"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>Deploy Command</Label>
                    <Textarea
                      value={deployCommand}
                      onChange={(event) => setDeployCommand(event.target.value)}
                      placeholder="pm2 restart ecosystem.config.js"
                      className="min-h-24 font-mono text-xs"
                    />
                  </div>
                </div>
              ) : null}

              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-1">
                <div className="space-y-2">
                  <Label>健康检查 URL</Label>
                  <Input
                    value={healthCheckUrl}
                    onChange={(event) => setHealthCheckUrl(event.target.value)}
                    placeholder="http://127.0.0.1:3000"
                  />
                </div>
                <div className="space-y-2">
                  <Label>自定义域名</Label>
                  <Input value={domain} onChange={(event) => setDomain(event.target.value)} placeholder="app.example.com" />
                </div>
                <div className="space-y-2 md:col-span-2 xl:col-span-1">
                  <Label>项目主域名</Label>
                  <Input
                    value={projectDomain}
                    onChange={(event) => setProjectDomain(event.target.value)}
                    placeholder="platform.example.com"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label>环境变量</Label>
                <Textarea
                  value={importEnv}
                  onChange={(event) => setImportEnv(event.target.value)}
                  placeholder={'NODE_ENV=production\nDATABASE_URL=postgres://...'}
                  className="min-h-28 font-mono text-xs"
                />
              </div>

              <label className="flex items-center gap-3 rounded-2xl border border-border/60 bg-background/80 px-4 py-3">
                <Checkbox
                  checked={autoDeployEnabled}
                  onCheckedChange={(checked) => setAutoDeployEnabled(checked === true)}
                />
                <div>
                  <div className="font-medium">开启自动部署</div>
                  <div className="text-xs text-muted-foreground">
                    保存 webhook secret，后续可以接 GitHub push webhook。
                  </div>
                </div>
              </label>

              <div className="rounded-2xl border border-border/60 bg-background/80 p-4 text-sm text-muted-foreground">
                <div className="flex items-center gap-2 font-medium text-foreground">
                  <PackageCheck className="h-4 w-4 text-primary" />
                  目标摘要
                </div>
                <div className="mt-3 space-y-1">
                  <div>Provider: {selectedTarget?.provider || 'shell'}</div>
                  <div>Workdir: {selectedTarget?.workingDirectory || '.'}</div>
                  <div>TLS: {selectedTarget?.certificateMode || 'disabled'}</div>
                </div>
              </div>

              <Button className="w-full" loading={importService.isPending} onClick={handleImportService}>
                <Rocket className="h-4 w-4" />
                导入为服务
              </Button>
            </div>
          </CardContent>
        </Card>
      </section>

      <section className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_440px]">
        <Card className="border-border/60 bg-card/95 shadow-lg">
          <CardHeader className="border-b border-border/50 bg-muted/20">
            <CardTitle className="flex items-center gap-2">
              <Server className="h-5 w-5 text-primary" />
              服务矩阵
            </CardTitle>
            <CardDescription>所有导入过的平台服务，以及最近一次部署状态。</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader className="bg-muted/20">
                <TableRow>
                  <TableHead>Service</TableHead>
                  <TableHead>Strategy</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Target</TableHead>
                  <TableHead className="text-right">Action</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {services.map((service) => {
                  const active = selectedServiceId === service.id;
                  return (
                    <TableRow
                      key={service.id}
                      className={cn('cursor-pointer hover:bg-muted/20', active && 'bg-primary/6')}
                      onClick={() => {
                        setSelectedServiceId(service.id);
                        if (service.lastDeploymentId) {
                          setSelectedDeploymentId(service.lastDeploymentId);
                        }
                      }}
                    >
                      <TableCell>
                        <div className="space-y-1">
                          <div className="font-medium">{service.name}</div>
                          <div className="text-xs text-muted-foreground">
                            {service.projectName} · {service.repositoryOwner}/{service.repositoryName}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Badge className={cn('rounded-full border-0', strategyTone(service.deployStrategy))}>
                          {service.deployStrategy}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge className={cn('rounded-full border', statusTone(service.lastDeployment?.status))}>
                          {service.lastDeployment?.status || 'idle'}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">{service.deployTarget}</TableCell>
                      <TableCell className="text-right">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={(event) => {
                            event.stopPropagation();
                            setSelectedServiceId(service.id);
                            if (service.lastDeploymentId) {
                              setSelectedDeploymentId(service.lastDeploymentId);
                            }
                          }}
                        >
                          <ChevronRight className="h-3.5 w-3.5" />
                          Open
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
                {services.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="py-10 text-center text-sm text-muted-foreground">
                      还没有服务。上面选个仓库导入一个。
                    </TableCell>
                  </TableRow>
                ) : null}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card className="border-border/60 bg-card/95 shadow-lg">
          <CardHeader className="border-b border-border/50 bg-muted/20">
            <CardTitle className="flex items-center gap-2">
              <Bot className="h-5 w-5 text-primary" />
              服务运维面板
            </CardTitle>
            <CardDescription>选中服务后，可编辑环境变量并追踪实时日志。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            {selectedService ? (
              <>
                <div className="rounded-[28px] border border-border/60 bg-[linear-gradient(135deg,rgba(16,185,129,0.08),rgba(14,165,233,0.05),transparent)] p-5">
                  <div className="flex items-start justify-between gap-4">
                    <div className="space-y-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <div className="text-xl font-semibold">{selectedService.name}</div>
                        <Badge className={cn('rounded-full border', statusTone(selectedService.lastDeployment?.status))}>
                          {selectedService.lastDeployment?.status || 'idle'}
                        </Badge>
                      </div>
                      <div className="flex flex-wrap items-center gap-3 text-xs text-muted-foreground">
                        <span className="inline-flex items-center gap-1">
                          <GitBranch className="h-3.5 w-3.5" />
                          {selectedService.defaultBranch}
                        </span>
                        <span className="inline-flex items-center gap-1">
                          <Globe className="h-3.5 w-3.5" />
                          {selectedService.domain || selectedServiceTarget?.domain || 'no domain'}
                        </span>
                        <span className="inline-flex items-center gap-1">
                          <TerminalSquare className="h-3.5 w-3.5" />
                          {selectedService.deployTarget}
                        </span>
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button loading={deployService.isPending} onClick={handleDeploySelectedService}>
                        <Rocket className="h-4 w-4" />
                        立即部署
                      </Button>
                      <Button variant="outline" asChild>
                        <a href={selectedService.repositoryUrl} target="_blank" rel="noreferrer">
                          <ExternalLink className="h-4 w-4" />
                          Repo
                        </a>
                      </Button>
                    </div>
                  </div>
                </div>

                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <Label>环境变量</Label>
                    <Button size="sm" variant="outline" loading={replaceEnvironment.isPending} onClick={handleSaveEnvironment}>
                      保存
                    </Button>
                  </div>
                  <Textarea
                    value={environmentEditor}
                    onChange={(event) => setEnvironmentEditor(event.target.value)}
                    className="min-h-40 font-mono text-xs"
                    placeholder={'NODE_ENV=production\nAPI_URL=https://api.example.com'}
                  />
                </div>

                <div className="space-y-3">
                  <div className="text-sm font-medium">最近部署</div>
                  <div className="space-y-2">
                    {deployments.map((deployment) => (
                      <button
                        key={deployment.id}
                        type="button"
                        onClick={() => setSelectedDeploymentId(deployment.id)}
                        className={cn(
                          'flex w-full items-center justify-between rounded-2xl border px-4 py-3 text-left transition',
                          selectedDeploymentId === deployment.id
                            ? 'border-primary/45 bg-primary/8'
                            : 'border-border/60 bg-muted/15 hover:border-primary/20'
                        )}
                      >
                        <div>
                          <div className="font-mono text-xs">{deployment.id.slice(0, 12)}</div>
                          <div className="mt-1 text-xs text-muted-foreground">
                            {formatTime(deployment.createdAt)}
                          </div>
                        </div>
                        <Badge className={cn('rounded-full border', statusTone(deployment.status))}>
                          {deployment.status}
                        </Badge>
                      </button>
                    ))}
                  </div>
                </div>

                <div className="space-y-2">
                  <div className="text-sm font-medium">实时日志</div>
                  <div
                    ref={logViewportRef}
                    className="max-h-[360px] overflow-auto rounded-[26px] border border-slate-900/90 bg-slate-950 p-4 font-mono text-xs leading-6 text-slate-100 shadow-inner"
                  >
                    {deferredLogs.length > 0 ? (
                      deferredLogs.map((entry) => (
                        <div key={entry.sequence} className="flex gap-3">
                          <span className="min-w-[76px] text-slate-500">
                            {new Date(entry.timestamp).toLocaleTimeString('en-IE', {
                              hour: '2-digit',
                              minute: '2-digit',
                              second: '2-digit',
                            })}
                          </span>
                          <span
                            className={cn(
                              'min-w-[54px] uppercase tracking-wide',
                              entry.stream === 'stderr' ? 'text-rose-300' : 'text-emerald-300'
                            )}
                          >
                            {entry.stream}
                          </span>
                          <span className="break-all text-slate-200">{entry.message}</span>
                        </div>
                      ))
                    ) : (
                      <div className="text-slate-500">选一个部署后，这里会开始滚动日志。</div>
                    )}
                  </div>
                </div>

                <div className="rounded-2xl border border-border/60 bg-muted/15 p-4 text-xs text-muted-foreground">
                  <div className="font-medium text-foreground">Webhook 与自动部署</div>
                  <div className="mt-2 space-y-1">
                    <div>状态: {selectedService.autoDeployEnabled ? 'Enabled' : 'Disabled'}</div>
                    <div>Secret: {selectedService.webhookSecret || 'Not generated'}</div>
                    <div>Endpoint: `/v1/platform/services/{selectedService.id}/webhooks/github` 预留</div>
                  </div>
                </div>
              </>
            ) : (
              <div className="rounded-2xl border border-dashed border-border/70 bg-muted/10 p-8 text-center text-sm text-muted-foreground">
                左边点一个服务，这里就会切到该服务的运维视图。
              </div>
            )}
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
