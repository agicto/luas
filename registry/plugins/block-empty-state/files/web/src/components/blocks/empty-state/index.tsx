/**
 * @block EmptyState
 * @description A centered state panel for empty collections, failed loads, missing access, and
 * empty search results. All copy and actions arrive through props; the block keeps no state and
 * fetches nothing, so it is safe in Server Components.
 */
import * as React from 'react';
import { Inbox, Lock, SearchX, TriangleAlert } from 'lucide-react';

import { cn } from '@/utils';

export type EmptyStateVariant = 'empty' | 'no-results' | 'error' | 'no-access';

export interface EmptyStateProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  variant?: EmptyStateVariant;
  /** Replaces the variant icon; it is decorative and hidden from assistive technology. */
  icon?: React.ReactNode;
  /** Main recovery action, for example a create or retry button. */
  action?: React.ReactNode;
  secondaryAction?: React.ReactNode;
  /** Technical detail such as a request id, shown small under the description. */
  detail?: React.ReactNode;
  headingLevel?: 'h2' | 'h3' | 'h4';
  /** Drops the dashed frame, for use inside an existing card or table cell. */
  bare?: boolean;
  className?: string;
}

const variantIcons: Record<EmptyStateVariant, React.ComponentType<{ className?: string }>> = {
  empty: Inbox,
  'no-results': SearchX,
  error: TriangleAlert,
  'no-access': Lock,
};

const iconTones: Record<EmptyStateVariant, string> = {
  empty: 'bg-muted text-muted-foreground',
  'no-results': 'bg-muted text-muted-foreground',
  error: 'bg-error-subtle text-error',
  'no-access': 'bg-warning-subtle text-warning',
};

export function EmptyState({
  title,
  description,
  variant = 'empty',
  icon,
  action,
  secondaryAction,
  detail,
  headingLevel = 'h2',
  bare = false,
  className,
}: EmptyStateProps) {
  const Heading = headingLevel;
  const Icon = variantIcons[variant];
  const headingId = React.useId();

  return (
    <section
      role={variant === 'error' ? 'alert' : undefined}
      aria-labelledby={headingId}
      data-variant={variant}
      className={cn(
        'flex flex-col items-center justify-center gap-4 px-4 py-10 text-center sm:px-6',
        !bare && 'rounded-xl border border-dashed bg-card',
        className
      )}
    >
      <span
        aria-hidden="true"
        className={cn(
          'grid size-12 place-items-center rounded-full [&>svg]:size-6',
          iconTones[variant]
        )}
      >
        {icon ?? <Icon />}
      </span>
      <div className="flex max-w-md flex-col gap-1.5">
        <Heading id={headingId} className="text-base font-semibold text-foreground">
          {title}
        </Heading>
        {description ? <p className="text-sm text-muted-foreground">{description}</p> : null}
        {detail ? (
          <p className="font-mono text-xs break-all text-muted-foreground">{detail}</p>
        ) : null}
      </div>
      {action || secondaryAction ? (
        <div className="flex flex-col-reverse items-stretch gap-2 sm:flex-row sm:items-center sm:justify-center">
          {secondaryAction}
          {action}
        </div>
      ) : null}
    </section>
  );
}
