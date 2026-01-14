import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"
import { Loader2 } from "lucide-react"

import { cn } from "@/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium focus-ring interactive shrink-0 disabled:cursor-not-allowed disabled:opacity-50",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground shadow-button-primary button-lighting hover:brightness-110 active:brightness-95",
        destructive:
          "bg-destructive text-white shadow-button-destructive button-destructive-lighting hover:bg-destructive/90 focus-visible:ring-destructive/30",
        outline:
          "border bg-background shadow-xs hover:bg-accent hover:text-accent-foreground dark:bg-input/30 dark:border-input dark:hover:bg-input/50",
        secondary:
          "bg-secondary text-secondary-foreground shadow-xs hover:bg-secondary-deeper",
        ghost:
          "hover:bg-secondary/80 hover:text-accent-foreground dark:hover:bg-accent/50",
        link: "text-muted-foreground underline-offset-4 hover:text-primary hover:underline",
      },
      size: {
        xs: "h-7 px-2 text-xs",
        sm: "h-8 px-3 text-xs",
        default: "h-9 px-4",
        lg: "h-10 px-6",
        xl: "h-11 px-8 text-base",
        "2xl": "h-12 px-10 text-lg",
      },
      isIcon: {
        true: "aspect-square p-0",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
      isIcon: false,
    },
  }
)

interface ButtonProps extends React.ComponentProps<"button">, VariantProps<typeof buttonVariants> {
  asChild?: boolean
  loading?: boolean
  icon?: React.ReactNode
  iconPosition?: "left" | "right"
}

function Button({
  className,
  variant,
  size,
  isIcon,
  loading = false,
  asChild = false,
  icon,
  iconPosition = "left",
  children,
  ...props
}: ButtonProps) {
  const Comp = asChild ? Slot : "button"

  const spinnerSize = cn(
    "animate-spin shrink-0",
    size === "xs" || size === "sm" ? "size-3" : 
    size === "xl" || size === "2xl" ? "size-6" : "size-4"
  )

  const spinner = <Loader2 className={spinnerSize} />

  return (
    <Comp
      data-slot="button"
      disabled={props.disabled || loading}
      className={cn(
        buttonVariants({ variant, size, isIcon, className }),
        // Dynamic rounding based on size
        size === "xs" || size === "sm" ? "rounded-md" : 
        size === "lg" || size === "xl" ? "rounded-xl" : 
        size === "2xl" ? "rounded-2xl" : "rounded-lg",
        loading && "relative pointer-events-none"
      )}
      {...props}
    >
      <span className={cn(
        "inline-flex items-center gap-2",
        isIcon ? "justify-center" : "justify-inherit"
      )}>
        {/* Left Slot: Show spinner if loading and position is left */}
        {iconPosition === "left" && (
          loading ? spinner : icon
        )}

        {/* Children Slot: Hidden only for isIcon buttons while loading */}
        {!(isIcon && loading) && children}

        {/* Right Slot: Show spinner if loading and position is right */}
        {iconPosition === "right" && (
          loading ? spinner : icon
        )}
      </span>
    </Comp>
  )
}

export { Button, buttonVariants }
