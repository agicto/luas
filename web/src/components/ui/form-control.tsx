import * as React from 'react';

import { cn } from '@/utils';

interface FormControlA11yOptions {
  id?: string;
  error?: boolean;
  errorText?: React.ReactNode;
  ariaDescribedBy?: string;
  ariaInvalid?: React.AriaAttributes['aria-invalid'];
}

function mergeDescriptionIds(...values: Array<string | undefined>): string | undefined {
  const ids = new Set(values.flatMap(value => value?.split(/\s+/).filter(Boolean) ?? []));

  return ids.size > 0 ? Array.from(ids).join(' ') : undefined;
}

export function useFormControlA11y({
  id,
  error = false,
  errorText,
  ariaDescribedBy,
  ariaInvalid,
}: FormControlA11yOptions) {
  const generatedId = React.useId();
  const hasErrorText = Boolean(errorText);
  const isInvalid = error || hasErrorText;
  const errorId = `${id ?? `form-control-${generatedId}`}-error`;

  return {
    errorId,
    isInvalid,
    ariaInvalid: isInvalid ? true : ariaInvalid,
    ariaDescribedBy: mergeDescriptionIds(ariaDescribedBy, hasErrorText ? errorId : undefined),
  };
}

export function FormControlError({ className, ...props }: React.ComponentProps<'p'>) {
  return (
    <p
      data-slot="form-control-error"
      aria-live="polite"
      className={cn(
        'mt-1 text-xs font-medium text-destructive animate-in fade-in slide-in-from-top-1 duration-200',
        className
      )}
      {...props}
    />
  );
}
