/**
 * @component Button
 * @category UI
 * @status Stable
 * @description A highly customizable button component with support for variants, sizes, loading states, and icons.
 * @usage Use for primary actions, forms, or navigation triggers. Supports 'asChild' for semantic flexibility.
 * @example
 * <Button variant="default" size="md" loading={isLoading} onClick={handleClick}>
 *   Click Me
 * </Button>
 */
import * as React from 'react';
import { Slot, Slottable } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { Loader2 } from 'lucide-react';

import { cn } from '@/utils';

const buttonVariants = cva(
  'inline-flex flex-row! flex-nowrap! items-center justify-center gap-2 whitespace-nowrap! rounded-md text-sm font-medium focus-ring shrink-0 disabled:cursor-not-allowed disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:cursor-not-allowed aria-disabled:opacity-50',
  {
    variants: {
      variant: {
        default:
          'bg-primary text-primary-foreground shadow-button-primary button-lighting hover:brightness-110 active:brightness-95',
        destructive:
          'bg-destructive text-white shadow-button-destructive button-destructive-lighting hover:bg-destructive/90 focus-visible:ring-destructive/30',
        outline:
          'border bg-background shadow-xs hover:bg-accent hover:text-accent-foreground dark:bg-input/30 dark:border-input dark:hover:bg-input/50',
        secondary: 'bg-secondary text-secondary-foreground shadow-xs hover:bg-secondary-deeper',
        ghost: 'hover:bg-secondary/80 hover:text-accent-foreground dark:hover:bg-accent/50',
        link: 'text-muted-foreground underline-offset-4 hover:text-primary hover:underline',
      },
      size: {
        xs: 'h-7 px-2 text-xs',
        sm: 'h-8 px-3 text-xs',
        default: 'h-9 px-4',
        lg: 'h-10 px-6',
        xl: 'h-11 px-8 text-base',
        '2xl': 'h-12 px-10 text-lg',
      },
      isIcon: {
        true: 'aspect-square p-0',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
      isIcon: false,
    },
  }
);

interface ButtonProps extends React.ComponentProps<'button'>, VariantProps<typeof buttonVariants> {
  /**
   * If true, the button will render as its child element while keeping button styles.
   */
  asChild?: boolean;
  /**
   * Displays a loading spinner and disables interaction.
   */
  loading?: boolean;
  /**
   * An optional icon to display inside the button.
   */
  icon?: React.ReactNode;
  /**
   * Positioning of the icon or loading spinner relative to children.
   * @default "left"
   */
  iconPosition?: 'left' | 'right';
  /**
   * Disables the scale transform on click. Use when Button is used as a DropdownMenuTrigger
   * to prevent dropdown jitter caused by transform conflicts.
   */
  noScale?: boolean;
}

function Button({
  className,
  variant,
  size,
  isIcon,
  loading = false,
  asChild = false,
  icon,
  iconPosition = 'left',
  noScale = false,
  children,
  disabled = false,
  tabIndex,
  onClick,
  onClickCapture,
  onKeyDown,
  onKeyDownCapture,
  'aria-busy': ariaBusy,
  'aria-disabled': ariaDisabled,
  ...props
}: ButtonProps) {
  const isDisabled = disabled || loading;

  const spinnerSize = cn(
    'animate-spin shrink-0',
    size === 'xs' || size === 'sm'
      ? 'size-3'
      : size === 'xl' || size === '2xl'
        ? 'size-6'
        : 'size-4'
  );

  const spinner = <Loader2 className={spinnerSize} aria-hidden="true" />;

  const leadingContent = iconPosition === 'left' ? (loading ? spinner : icon) : null;
  const trailingContent = iconPosition === 'right' ? (loading ? spinner : icon) : null;
  const composedChild = prepareComposedChild(
    children,
    isDisabled,
    loading,
    Boolean(isIcon && loading)
  );
  const nativeContent = (
    <>
      {leadingContent}
      {!(isIcon && loading) && children}
      {trailingContent}
    </>
  );
  const resolvedClassName = cn(
    buttonVariants({ variant, size, isIcon, className }),
    noScale ? 'interactive-no-scale' : 'interactive',
    size === 'xs' || size === 'sm'
      ? 'rounded-md'
      : size === 'lg' || size === 'xl'
        ? 'rounded-xl'
        : size === '2xl'
          ? 'rounded-2xl'
          : 'rounded-lg',
    loading && 'relative'
  );

  if (asChild) {
    return (
      <Slot
        data-slot="button"
        data-disabled={isDisabled ? '' : undefined}
        data-loading={loading ? '' : undefined}
        aria-busy={loading ? true : ariaBusy}
        aria-disabled={isDisabled ? true : ariaDisabled}
        tabIndex={isDisabled ? -1 : tabIndex}
        className={resolvedClassName}
        onClick={isDisabled ? undefined : asElementMouseHandler(onClick)}
        onClickCapture={isDisabled ? undefined : asElementMouseHandler(onClickCapture)}
        onKeyDown={isDisabled ? undefined : asElementKeyboardHandler(onKeyDown)}
        onKeyDownCapture={isDisabled ? undefined : asElementKeyboardHandler(onKeyDownCapture)}
        {...props}
      >
        {leadingContent}
        <Slottable>{composedChild}</Slottable>
        {trailingContent}
      </Slot>
    );
  }

  return (
    <button
      data-slot="button"
      data-disabled={isDisabled ? '' : undefined}
      data-loading={loading ? '' : undefined}
      disabled={isDisabled}
      aria-busy={loading ? true : ariaBusy}
      aria-disabled={ariaDisabled}
      tabIndex={tabIndex}
      className={resolvedClassName}
      onClick={onClick}
      onClickCapture={onClickCapture}
      onKeyDown={onKeyDown}
      onKeyDownCapture={onKeyDownCapture}
      {...props}
    >
      {nativeContent}
    </button>
  );
}

const preventDisabledClick: React.MouseEventHandler<HTMLElement> = event => {
  event.preventDefault();
  event.stopPropagation();
};

const preventDisabledKeyDown: React.KeyboardEventHandler<HTMLElement> = event => {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault();
    event.stopPropagation();
  }
};

function prepareComposedChild(
  children: React.ReactNode,
  isDisabled: boolean,
  loading: boolean,
  hideContent: boolean
): React.ReactNode {
  if (!React.isValidElement<React.HTMLAttributes<HTMLElement>>(children)) {
    return children;
  }

  if (!isDisabled && !loading && !hideContent) {
    return children;
  }

  return React.cloneElement(
    children,
    {
      ...(isDisabled
        ? {
            'aria-disabled': true,
            tabIndex: -1,
            onClick: undefined,
            onClickCapture: preventDisabledClick,
            onKeyDown: undefined,
            onKeyDownCapture: preventDisabledKeyDown,
          }
        : undefined),
      ...(loading ? { 'aria-busy': true } : undefined),
    },
    hideContent ? null : children.props.children
  );
}

function asElementMouseHandler(
  handler: React.MouseEventHandler<HTMLButtonElement> | undefined
): React.MouseEventHandler<HTMLElement> | undefined {
  return handler as React.MouseEventHandler<HTMLElement> | undefined;
}

function asElementKeyboardHandler(
  handler: React.KeyboardEventHandler<HTMLButtonElement> | undefined
): React.KeyboardEventHandler<HTMLElement> | undefined {
  return handler as React.KeyboardEventHandler<HTMLElement> | undefined;
}

export { Button, buttonVariants };
