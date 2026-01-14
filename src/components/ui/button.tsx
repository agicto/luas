import * as React from "react"
import { Slot } from "@radix-ui/react-slot"
import { cva, type VariantProps } from "class-variance-authority"
import { Loader2 } from "lucide-react"

import { cn } from "@/utils"

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium focus-ring interactive [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default:
          "bg-primary text-primary-foreground shadow-xs hover:bg-primary/90",
        destructive:
          "bg-destructive text-white shadow-xs hover:bg-destructive/90 focus-visible:ring-destructive/20 dark:focus-visible:ring-destructive/40 dark:bg-destructive/60",
        outline:
          "border bg-background shadow-xs hover:bg-accent hover:text-accent-foreground dark:bg-input/30 dark:border-input dark:hover:bg-input/50",
        secondary:
          "bg-secondary text-secondary-foreground shadow-xs hover:bg-secondary/80",
        ghost:
          "hover:bg-accent hover:text-accent-foreground dark:hover:bg-accent/50",
        link: "text-primary underline-offset-4 hover:underline",
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

  const spinner = (
    <Loader2 
      className={cn(
        "animate-spin shrink-0",
        size === "xs" || size === "sm" ? "size-3" : 
        size === "xl" || size === "2xl" ? "size-6" : "size-4"
      )} 
    />
  )

  const iconToRender = loading ? spinner : icon
  const showIcon = icon || loading

  return (
    <Comp
      data-slot="button"
      disabled={props.disabled || loading}
      className={cn(
        buttonVariants({ variant, size, isIcon, className }),
        loading && !icon && !isIcon && "relative pointer-events-none"
      )}
      {...props}
    >
      {loading && !icon && !isIcon && (
        <div className="absolute inset-0 flex items-center justify-center bg-background/20 dark:bg-black/10 z-10 transition-all duration-200">
          {spinner}
        </div>
      )}
      
      <span className={cn(
        "inline-flex items-center gap-2",
        isIcon ? "justify-center" : "justify-inherit",
        loading && !icon && !isIcon && "opacity-0"
      )}>
        {showIcon && iconPosition === "left" && iconToRender}
        {children}
        {showIcon && iconPosition === "right" && iconToRender}
      </span>
    </Comp>
  )
}

export { Button, buttonVariants }
