import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { SearchIcon, EyeIcon, EyeOffIcon, AlertCircleIcon } from "lucide-react"

import { cn } from "@/utils"

const inputVariants = cva(
  "file:text-foreground placeholder:text-muted-foreground selection:bg-primary selection:text-primary-foreground flex h-9 w-full min-w-0 rounded-md px-3 py-1 text-base shadow-xs transition-all file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium disabled:cursor-not-allowed disabled:opacity-50 md:text-sm focus-border",
  {
    variants: {
      variant: {
        outline: "border border-input bg-transparent dark:bg-input/30",
        filled: "border-transparent bg-muted/50 hover:bg-muted focus:bg-background focus:border-input",
      },
      error: {
        true: "border-destructive focus-visible:border-destructive text-destructive placeholder:text-destructive/50",
      }
    },
    defaultVariants: {
      variant: "outline",
    },
  }
)

interface InputProps extends React.ComponentProps<"input">, VariantProps<typeof inputVariants> {
  leftIcon?: React.ReactNode
  rightIcon?: React.ReactNode
  error?: boolean
  errorText?: React.ReactNode
  root?: boolean
}

function Input({
  className,
  variant,
  type,
  leftIcon,
  rightIcon,
  error,
  errorText,
  root,
  ...props
}: InputProps) {
  const isError = error || !!errorText

  const inputNode = (
    <input
      type={type}
      data-slot="input"
      className={cn(
        inputVariants({ variant, error: isError, className }),
        leftIcon && "pl-10",
        rightIcon && "pr-10"
      )}
      {...props}
    />
  )

  const withIcons = (leftIcon || rightIcon) ? (
    <div className="relative flex w-full items-center">
      {leftIcon && (
        <div className="absolute left-3 flex items-center justify-center text-muted-foreground pointer-events-none">
          {leftIcon}
        </div>
      )}
      {inputNode}
      {rightIcon && (
        <div className="absolute right-3 flex items-center justify-center text-muted-foreground">
          {rightIcon}
        </div>
      )}
    </div>
  ) : inputNode

  if (root || errorText) {
    return (
      <div className="flex flex-col gap-0.5 w-full">
        {withIcons}
        {errorText && (
          <p className="text-xs font-medium text-destructive animate-in fade-in slide-in-from-top-1 duration-200">
            {errorText}
          </p>
        )}
      </div>
    )
  }

  return withIcons
}

function SearchInput({ className, ...props }: InputProps) {
  return (
    <Input
      type="search"
      leftIcon={<SearchIcon className="size-4" />}
      className={cn("rounded-full", className)}
      {...props}
    />
  )
}

function PasswordInput({ className, ...props }: InputProps) {
  const [show, setShow] = React.useState(false)

  return (
    <Input
      type={show ? "text" : "password"}
      rightIcon={
        <button
          type="button"
          onClick={() => setShow(!show)}
          className="hover:text-foreground cursor-pointer transition-colors outline-hidden focus-visible:ring-1 focus-visible:ring-ring rounded-sm pointer-events-auto"
        >
          {show ? <EyeOffIcon className="size-4" /> : <EyeIcon className="size-4" />}
        </button>
      }
      className={className}
      {...props}
    />
  )
}

export { Input, SearchInput, PasswordInput }
