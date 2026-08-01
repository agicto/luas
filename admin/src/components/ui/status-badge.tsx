import type { HTMLAttributes } from 'react';
import { cn } from '@/lib/cn';

type StatusTone = 'danger' | 'neutral' | 'success' | 'warning';

const toneClasses: Record<StatusTone, string> = {
  danger: 'border-danger/25 bg-danger-subtle text-danger-strong',
  neutral: 'border-border bg-muted text-subtle',
  success: 'border-success/25 bg-success-subtle text-success-strong',
  warning: 'border-warning/30 bg-warning-subtle text-warning-strong',
};

interface StatusBadgeProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: StatusTone;
}

export function StatusBadge({ className, children, tone = 'neutral', ...props }: StatusBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex h-6 items-center gap-1.5 rounded-full border px-2 text-xs font-medium',
        toneClasses[tone],
        className,
      )}
      {...props}
    >
      <span className="size-1.5 rounded-full bg-current" aria-hidden="true" />
      {children}
    </span>
  );
}
