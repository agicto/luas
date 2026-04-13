'use client';

import * as React from 'react';
import {
  Activity,
  CheckCircle2,
  Clock3,
  FileKey2,
  Globe,
  Rocket,
  Server,
  ShieldCheck,
  Terminal,
} from 'lucide-react';
import { toast } from 'sonner';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
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
  useDeployment,
  useDeploymentLogs,
  useDeployments,
  useDeployTargets,
  useGenerateCertificate,
  useStartDeployment,
} from '@/features/deploy/hooks/use-deploy';
import { deployService } from '@/features/deploy/services/deploy-service';
import type {
  Deployment,
  DeploymentLogEntry,
  DeploymentWatchEvent,
} from '@/features/deploy/types';

function statusTone(status: Deployment['status']) {
  switch (status) {
    case 'succeeded':
      return 'border-emerald-500/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300';
    case 'failed':
      return 'border-rose-500/20 bg-rose-500/10 text-rose-700 dark:text-rose-300';
    case 'running':
      return 'border-sky-500/20 bg-sky-500/10 text-sky-700 dark:text-sky-300';
    default:
      return 'border-amber-500/20 bg-amber-500/10 text-amber-700 dark:text-amber-300';
  }
}

function parseEnvironmentInput(source: string): Record<string, string> {
  const entries = source
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);

  const env: Record<string, string> = {};
  for (const entry of entries) {
    const parts = entry.split('=');
    if (parts.length < 2) {
      throw new Error(`Invalid environment line: ${entry}`);
    }
    const key = parts[0]?.trim();
    const value = parts.slice(1).join('=').trim();
    if (!key) {
      throw new Error(`Environment key is missing: ${entry}`);
    }
    env[key] = value;
  }

  return env;
}

function formatTime(value?: string) {
  if (!value) {
    return 'Pending';
  }

  return new Intl.DateTimeFormat('en-IE', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(new Date(value));
}

export default function DeploymentsPage() {
  const { data: targets = [], isLoading: isTargetsLoading } = useDeployTargets();
  const { data: deployments = [], refetch: refetchDeployments } = useDeployments(30);

  const [selectedTarget, setSelectedTarget] = React.useState('');
  const [selectedDeploymentId, setSelectedDeploymentId] = React.useState<string | null>(null);
  const [branch, setBranch] = React.useState('main');
  const [commit, setCommit] = React.useState('');
  const [envText, setEnvText] = React.useState('');
  const [certificateDomain, setCertificateDomain] = React.useState('');
  const [certificateDays, setCertificateDays] = React.useState('90');
  const [streamedDeployment, setStreamedDeployment] = React.useState<Deployment | null>(null);
  const [streamedLogs, setStreamedLogs] = React.useState<DeploymentLogEntry[]>([]);

  const { data: deploymentDetail } = useDeployment(selectedDeploymentId);
  const { data: deploymentLogs = [] } = useDeploymentLogs(selectedDeploymentId, 300);
  const startDeployment = useStartDeployment();
  const generateCertificate = useGenerateCertificate();

  const selectedTargetConfig = targets.find((target) => target.name === selectedTarget) ?? null;
  const activeDeployment = streamedDeployment ?? deploymentDetail ?? null;
  const deferredLogs = React.useDeferredValue(streamedLogs);
  const logViewportRef = React.useRef<HTMLDivElement | null>(null);

  React.useEffect(() => {
    if (!selectedTarget && targets.length > 0) {
      setSelectedTarget(targets[0].name);
      setCertificateDomain(targets[0].domain ?? '');
    }
  }, [selectedTarget, targets]);

  React.useEffect(() => {
    if (!selectedDeploymentId && deployments.length > 0) {
      setSelectedDeploymentId(deployments[0].id);
    }
  }, [deployments, selectedDeploymentId]);

  React.useEffect(() => {
    setStreamedDeployment(deploymentDetail ?? null);
  }, [deploymentDetail]);

  React.useEffect(() => {
    setStreamedLogs(deploymentLogs);
  }, [deploymentLogs, selectedDeploymentId]);

  React.useEffect(() => {
    if (selectedTargetConfig?.domain) {
      setCertificateDomain(selectedTargetConfig.domain);
    }
  }, [selectedTargetConfig?.domain]);

  React.useEffect(() => {
    if (!selectedDeploymentId) {
      return;
    }

    const eventSource = new EventSource(deployService.streamUrl(selectedDeploymentId), {
      withCredentials: true,
    });

    eventSource.onmessage = (message) => {
      try {
        const payload = JSON.parse(message.data) as DeploymentWatchEvent;

        if (payload.log) {
          React.startTransition(() => {
            setStreamedLogs((current) => {
              if (current.some((entry) => entry.sequence === payload.log!.sequence)) {
                return current;
              }
              const next = [...current, payload.log!];
              return next.slice(-400);
            });
          });
        }

        if (payload.deployment) {
          setStreamedDeployment(payload.deployment);
        }

        if (payload.done) {
          void refetchDeployments();
          eventSource.close();
        }
      } catch {
        // Ignore malformed event payloads.
      }
    };

    eventSource.onerror = () => {
      eventSource.close();
    };

    return () => {
      eventSource.close();
    };
  }, [refetchDeployments, selectedDeploymentId]);

  React.useEffect(() => {
    if (!logViewportRef.current) {
      return;
    }

    logViewportRef.current.scrollTop = logViewportRef.current.scrollHeight;
  }, [deferredLogs]);

  const runningCount = deployments.filter((deployment) => deployment.status === 'running').length;
  const successCount = deployments.filter((deployment) => deployment.status === 'succeeded').length;
  const failureCount = deployments.filter((deployment) => deployment.status === 'failed').length;
  const successRate =
    deployments.length === 0 ? 0 : Math.round((successCount / deployments.length) * 100);

  async function handleStartDeployment() {
    if (!selectedTarget) {
      toast.error('Select a deployment target first.');
      return;
    }

    let environment: Record<string, string> | undefined;
    try {
      environment = envText.trim() ? parseEnvironmentInput(envText) : undefined;
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Invalid environment variables.');
      return;
    }

    const result = await startDeployment.mutateAsync({
      target: selectedTarget,
      branch: branch.trim() || undefined,
      commit: commit.trim() || undefined,
      triggeredBy: 'console',
      environment,
    });

    setSelectedDeploymentId(result.deployment.id);
    setStreamedDeployment(result.deployment);
    setStreamedLogs([]);
    toast.success(`Deployment ${result.deployment.id.slice(0, 8)} queued.`);
  }

  async function handleGenerateCertificate() {
    if (!certificateDomain.trim()) {
      toast.error('Enter a domain before generating a certificate.');
      return;
    }

    const validDays = Number.parseInt(certificateDays, 10);
    const result = await generateCertificate.mutateAsync({
      domain: certificateDomain.trim(),
      validDays: Number.isNaN(validDays) ? 90 : validDays,
    });

    toast.success(`Certificate generated at ${result.certPath}`);
  }

  return (
    <div className="flex-1 space-y-6 p-6 pt-6">
      <div className="relative overflow-hidden rounded-3xl border border-border/60 bg-[radial-gradient(circle_at_top_left,_rgba(16,185,129,0.18),_transparent_34%),radial-gradient(circle_at_bottom_right,_rgba(14,165,233,0.18),_transparent_28%),linear-gradient(135deg,_rgba(15,23,42,0.98),_rgba(17,24,39,0.96))] p-6 text-white shadow-2xl">
        <div className="absolute inset-0 opacity-30 [background-image:linear-gradient(rgba(255,255,255,0.08)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.08)_1px,transparent_1px)] [background-size:28px_28px]" />
        <div className="relative flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div className="space-y-3">
            <Badge className="rounded-full border border-white/15 bg-white/10 px-3 py-1 text-white">
              Release Orchestration
            </Badge>
            <div className="space-y-2">
              <h1 className="text-3xl font-semibold tracking-tight">Automated Deploy Control</h1>
              <p className="max-w-2xl text-sm text-white/70">
                Trigger releases from the console, stream runtime logs live, and manage TLS
                certificates from the same operational surface.
              </p>
            </div>
          </div>
          <div className="grid gap-3 sm:grid-cols-3">
            <div className="rounded-2xl border border-white/10 bg-white/8 px-4 py-3 backdrop-blur">
              <div className="text-xs uppercase tracking-[0.24em] text-white/50">Running</div>
              <div className="mt-2 text-2xl font-semibold">{runningCount}</div>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/8 px-4 py-3 backdrop-blur">
              <div className="text-xs uppercase tracking-[0.24em] text-white/50">Success Rate</div>
              <div className="mt-2 text-2xl font-semibold">{successRate}%</div>
            </div>
            <div className="rounded-2xl border border-white/10 bg-white/8 px-4 py-3 backdrop-blur">
              <div className="text-xs uppercase tracking-[0.24em] text-white/50">Failures</div>
              <div className="mt-2 text-2xl font-semibold">{failureCount}</div>
            </div>
          </div>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[420px_minmax(0,1fr)]">
        <Card className="overflow-hidden border-border/60 bg-card/95 shadow-lg">
          <CardHeader className="border-b border-border/50 bg-muted/20">
            <CardTitle className="flex items-center gap-2">
              <Rocket className="h-5 w-5 text-primary" />
              Release Trigger
            </CardTitle>
            <CardDescription>Choose a target, attach release metadata, then run it immediately.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="target">Deployment Target</Label>
              <Select value={selectedTarget} onValueChange={setSelectedTarget}>
                <SelectTrigger id="target" className="w-full">
                  <SelectValue placeholder={isTargetsLoading ? 'Loading targets...' : 'Select target'} />
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

            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-1">
              <div className="space-y-2">
                <Label htmlFor="branch">Branch</Label>
                <Input id="branch" value={branch} onChange={(event) => setBranch(event.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="commit">Commit</Label>
                <Input
                  id="commit"
                  placeholder="Optional commit SHA"
                  value={commit}
                  onChange={(event) => setCommit(event.target.value)}
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="env">Environment Overrides</Label>
              <Textarea
                id="env"
                placeholder={'RELEASE_VERSION=2026.04.10\nNEXT_PUBLIC_APP_URL=https://app.example.com'}
                value={envText}
                onChange={(event) => setEnvText(event.target.value)}
              />
            </div>

            {selectedTargetConfig ? (
              <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-sm font-medium">{selectedTargetConfig.displayName}</div>
                    <div className="mt-1 text-xs text-muted-foreground">
                      {selectedTargetConfig.workingDirectory}
                    </div>
                  </div>
                  <Badge variant="outline" className="rounded-full">
                    {selectedTargetConfig.provider}
                  </Badge>
                </div>
                <div className="mt-4 space-y-2 text-xs text-muted-foreground">
                  <div className="flex items-center gap-2">
                    <Server className="h-3.5 w-3.5" />
                    <span>{selectedTargetConfig.deployCommand}</span>
                  </div>
                  {selectedTargetConfig.healthCheckUrl ? (
                    <div className="flex items-center gap-2">
                      <Activity className="h-3.5 w-3.5" />
                      <span>{selectedTargetConfig.healthCheckUrl}</span>
                    </div>
                  ) : null}
                  {selectedTargetConfig.domain ? (
                    <div className="flex items-center gap-2">
                      <Globe className="h-3.5 w-3.5" />
                      <span>{selectedTargetConfig.domain}</span>
                    </div>
                  ) : null}
                </div>
              </div>
            ) : null}

            <Button
              className="h-11 w-full gap-2 rounded-2xl"
              onClick={() => void handleStartDeployment()}
              disabled={startDeployment.isPending || !selectedTarget}
            >
              <Rocket className="h-4 w-4" />
              {startDeployment.isPending ? 'Queueing deployment...' : 'Run Deployment'}
            </Button>
          </CardContent>
        </Card>

        <Card className="overflow-hidden border-border/60 shadow-lg">
          <CardHeader className="border-b border-border/50 bg-muted/15">
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Terminal className="h-5 w-5 text-primary" />
                  Live Deployment Logs
                </CardTitle>
                <CardDescription>
                  {activeDeployment
                    ? `Streaming ${activeDeployment.targetName} (${activeDeployment.id.slice(0, 8)})`
                    : 'Select a deployment to inspect live output.'}
                </CardDescription>
              </div>
              {activeDeployment ? (
                <Badge
                  variant="outline"
                  className={cn('rounded-full border px-3 py-1 text-xs capitalize', statusTone(activeDeployment.status))}
                >
                  {activeDeployment.status}
                </Badge>
              ) : null}
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-3 md:grid-cols-3">
              <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
                <div className="text-xs uppercase tracking-[0.2em] text-muted-foreground">Target</div>
                <div className="mt-2 text-sm font-medium">{activeDeployment?.targetName ?? 'None selected'}</div>
              </div>
              <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
                <div className="text-xs uppercase tracking-[0.2em] text-muted-foreground">Started</div>
                <div className="mt-2 text-sm font-medium">{formatTime(activeDeployment?.startedAt)}</div>
              </div>
              <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
                <div className="text-xs uppercase tracking-[0.2em] text-muted-foreground">TLS</div>
                <div className="mt-2 text-sm font-medium">{activeDeployment?.certificateMode ?? 'disabled'}</div>
              </div>
            </div>

            <div
              ref={logViewportRef}
              className="h-[420px] overflow-y-auto rounded-3xl border border-slate-800 bg-slate-950 p-4 font-mono text-xs text-slate-100 shadow-inner"
            >
              {deferredLogs.length === 0 ? (
                <div className="flex h-full items-center justify-center text-slate-500">
                  Waiting for deployment output...
                </div>
              ) : (
                <div className="space-y-2">
                  {deferredLogs.map((entry) => (
                    <div key={entry.sequence} className="grid grid-cols-[68px_60px_minmax(0,1fr)] gap-3">
                      <span className="text-slate-500">
                        {new Date(entry.timestamp).toLocaleTimeString('en-IE', {
                          hour: '2-digit',
                          minute: '2-digit',
                          second: '2-digit',
                        })}
                      </span>
                      <span
                        className={cn(
                          'uppercase tracking-wide',
                          entry.stream === 'stderr' ? 'text-rose-300' : 'text-emerald-300'
                        )}
                      >
                        {entry.stream}
                      </span>
                      <span className="break-words text-slate-200">{entry.message}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {activeDeployment?.error ? (
              <div className="rounded-2xl border border-rose-500/20 bg-rose-500/10 px-4 py-3 text-sm text-rose-700 dark:text-rose-300">
                {activeDeployment.error}
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_360px]">
        <Card className="overflow-hidden border-border/60 shadow-lg">
          <CardHeader className="border-b border-border/50 bg-muted/15">
            <CardTitle className="flex items-center gap-2">
              <Clock3 className="h-5 w-5 text-primary" />
              Deployment History
            </CardTitle>
            <CardDescription>
              Recent releases from CLI, webhook, and the web console are all tracked here.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow className="border-border/50">
                  <TableHead className="pl-6">Target</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Trigger</TableHead>
                  <TableHead>Time</TableHead>
                  <TableHead className="pr-6 text-right">Inspect</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {deployments.map((deployment) => (
                  <TableRow
                    key={deployment.id}
                    className="cursor-pointer border-border/40"
                    onClick={() => setSelectedDeploymentId(deployment.id)}
                  >
                    <TableCell className="pl-6">
                      <div className="space-y-1">
                        <div className="font-medium">{deployment.targetName}</div>
                        <div className="font-mono text-xs text-muted-foreground">
                          {deployment.id.slice(0, 8)}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant="outline"
                        className={cn('rounded-full border px-3 py-1 capitalize', statusTone(deployment.status))}
                      >
                        {deployment.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm">{deployment.triggerMode}</div>
                      <div className="text-xs text-muted-foreground">{deployment.triggeredBy}</div>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {formatTime(deployment.createdAt)}
                    </TableCell>
                    <TableCell className="pr-6 text-right">
                      <Button variant="ghost" size="sm" className="rounded-full">
                        View
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>

        <Card className="overflow-hidden border-border/60 shadow-lg">
          <CardHeader className="border-b border-border/50 bg-muted/15">
            <CardTitle className="flex items-center gap-2">
              <FileKey2 className="h-5 w-5 text-primary" />
              TLS Toolkit
            </CardTitle>
            <CardDescription>
              Generate a self-signed bundle for self-hosted environments, or keep Render on managed TLS.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
              <div className="flex items-center gap-2 text-sm font-medium">
                <ShieldCheck className="h-4 w-4 text-primary" />
                Certificate Mode
              </div>
              <div className="mt-2 text-sm text-muted-foreground">
                Current target: {selectedTargetConfig?.certificateMode ?? 'disabled'}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="cert-domain">Domain</Label>
              <Input
                id="cert-domain"
                value={certificateDomain}
                onChange={(event) => setCertificateDomain(event.target.value)}
                placeholder="app.example.com"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="cert-days">Valid Days</Label>
              <Input
                id="cert-days"
                value={certificateDays}
                onChange={(event) => setCertificateDays(event.target.value)}
                placeholder="90"
              />
            </div>

            <Button
              variant="outline"
              className="h-11 w-full rounded-2xl"
              onClick={() => void handleGenerateCertificate()}
              disabled={generateCertificate.isPending}
            >
              {generateCertificate.isPending ? 'Generating certificate...' : 'Generate Self-Signed Certificate'}
            </Button>

            <div className="rounded-2xl border border-dashed border-border/60 bg-muted/10 p-4 text-sm text-muted-foreground">
              Render targets should stay on <span className="font-medium text-foreground">render-managed</span>.
              Self-signed output is intended for reverse proxies or private environments.
            </div>

            {activeDeployment?.certificate ? (
              <div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/10 p-4 text-sm">
                <div className="font-medium text-emerald-700 dark:text-emerald-300">
                  Latest generated certificate
                </div>
                <div className="mt-2 break-all text-muted-foreground">{activeDeployment.certificate.certPath}</div>
                <div className="mt-1 break-all text-muted-foreground">{activeDeployment.certificate.keyPath}</div>
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card className="border-border/60 bg-muted/15">
          <CardContent className="flex items-center gap-4 py-6">
            <div className="rounded-2xl bg-primary/10 p-3 text-primary">
              <CheckCircle2 className="h-5 w-5" />
            </div>
            <div>
              <div className="text-sm text-muted-foreground">Successful runs</div>
              <div className="text-2xl font-semibold">{successCount}</div>
            </div>
          </CardContent>
        </Card>
        <Card className="border-border/60 bg-muted/15">
          <CardContent className="flex items-center gap-4 py-6">
            <div className="rounded-2xl bg-sky-500/10 p-3 text-sky-600 dark:text-sky-300">
              <Activity className="h-5 w-5" />
            </div>
            <div>
              <div className="text-sm text-muted-foreground">Tracked targets</div>
              <div className="text-2xl font-semibold">{targets.length}</div>
            </div>
          </CardContent>
        </Card>
        <Card className="border-border/60 bg-muted/15">
          <CardContent className="flex items-center gap-4 py-6">
            <div className="rounded-2xl bg-amber-500/10 p-3 text-amber-600 dark:text-amber-300">
              <Globe className="h-5 w-5" />
            </div>
            <div>
              <div className="text-sm text-muted-foreground">Deploy API</div>
              <div className="text-sm font-medium">Configured from environment</div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
