/**
 * @block OnboardingSteps
 * @description An onboarding checklist: a progress bar and an ordered list of steps that are
 * complete, current, or upcoming, each with an optional action. Step status comes from the
 * caller; the block keeps no state and fetches nothing, so it is safe in Server Components. All
 * copy arrives through props, including the formatted progress label.
 */
import * as React from 'react';
import { Circle, CircleCheck, CircleDot } from 'lucide-react';

import { Card, CardContent, CardDescription, CardHeader } from '@/components/ui/card';
import { cn } from '@/utils';

export type OnboardingStepStatus = 'complete' | 'current' | 'upcoming';

export interface OnboardingStep {
  id: string;
  title: React.ReactNode;
  description?: React.ReactNode;
  status: OnboardingStepStatus;
  /** Shown on the current step, for example a button or link that starts the step. */
  action?: React.ReactNode;
}

export interface OnboardingStepsLabels {
  /** Formatted progress, for example "2 of 5 steps complete". */
  progress: string;
  /** Announced after each step title, for example "Completed", "Current step", "Not started". */
  complete: string;
  current: string;
  upcoming: string;
}

export interface OnboardingStepsProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  steps: OnboardingStep[];
  labels: OnboardingStepsLabels;
  headingLevel?: 'h2' | 'h3';
  className?: string;
}

const statusIcons: Record<OnboardingStepStatus, React.ComponentType<{ className?: string }>> = {
  complete: CircleCheck,
  current: CircleDot,
  upcoming: Circle,
};

const statusTones: Record<OnboardingStepStatus, string> = {
  complete: 'text-success',
  current: 'text-primary',
  upcoming: 'text-muted-foreground',
};

export function OnboardingSteps({
  title,
  description,
  steps,
  labels,
  headingLevel = 'h2',
  className,
}: OnboardingStepsProps) {
  const Heading = headingLevel;
  const StepHeading = headingLevel === 'h2' ? 'h3' : 'h4';
  const headingId = React.useId();
  const completed = steps.filter((step) => step.status === 'complete').length;
  const percent = steps.length === 0 ? 0 : Math.round((completed / steps.length) * 100);

  return (
    <Card role="region" aria-labelledby={headingId} className={cn('gap-5', className)}>
      <CardHeader className="gap-3">
        <Heading id={headingId} className="text-lg font-semibold text-card-foreground">
          {title}
        </Heading>
        {description ? <CardDescription>{description}</CardDescription> : null}
        <div className="flex flex-col gap-2">
          <div
            role="progressbar"
            aria-label={labels.progress}
            aria-valuemin={0}
            aria-valuemax={steps.length}
            aria-valuenow={completed}
            className="h-2 w-full overflow-hidden rounded-full bg-muted"
          >
            <div
              className="h-full rounded-full bg-primary transition-[width] duration-300 motion-reduce:transition-none"
              style={{ width: `${percent}%` }}
            />
          </div>
          <p className="text-sm text-muted-foreground" aria-hidden="true">
            {labels.progress}
          </p>
        </div>
      </CardHeader>
      <CardContent>
        <ol className="flex flex-col">
          {steps.map((step, index) => {
            const Icon = statusIcons[step.status];
            const last = index === steps.length - 1;
            return (
              <li
                key={step.id}
                aria-current={step.status === 'current' ? 'step' : undefined}
                data-status={step.status}
                className="relative flex gap-3 pb-5 last:pb-0"
              >
                {!last ? (
                  <span
                    aria-hidden="true"
                    className={cn(
                      'absolute top-7 bottom-1 left-3 w-px -translate-x-1/2',
                      step.status === 'complete' ? 'bg-success' : 'bg-border'
                    )}
                  />
                ) : null}
                <Icon className={cn('mt-0.5 size-6 shrink-0', statusTones[step.status])} aria-hidden="true" />
                <div className="flex min-w-0 flex-1 flex-col gap-1">
                  <StepHeading
                    className={cn(
                      'text-sm font-medium',
                      step.status === 'upcoming' ? 'text-muted-foreground' : 'text-card-foreground'
                    )}
                  >
                    {step.title}
                    <span className="sr-only"> ({labels[step.status]})</span>
                  </StepHeading>
                  {step.description ? (
                    <p className="text-sm text-muted-foreground">{step.description}</p>
                  ) : null}
                  {step.status === 'current' && step.action ? (
                    <div className="mt-2 flex flex-wrap gap-2">{step.action}</div>
                  ) : null}
                </div>
              </li>
            );
          })}
        </ol>
      </CardContent>
    </Card>
  );
}
