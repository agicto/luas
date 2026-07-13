/**
 * @component Input
 * @category UI
 * @status Stable
 * @description A standard input field with support for icons, error states, and specialized types like search, password, and color.
 * @usage Use for native HTML input semantics. Import DatePicker or ColorPicker explicitly when a custom control is required.
 * @example
 * <Input placeholder="Username" leftIcon={<User />} errorText="Username is required" />
 * <PasswordInput placeholder="Password" showPasswordLabel="Show password" hidePasswordLabel="Hide password" />
 */
import * as React from "react"
import { cva, type VariantProps } from "class-variance-authority"
import { SearchIcon, EyeIcon, EyeOffIcon } from "lucide-react"

import { cn } from "@/utils"
import { FormControlError, useFormControlA11y } from "./form-control"

const inputVariants = cva(
  "file:text-foreground placeholder:text-muted-foreground selection:bg-primary selection:text-primary-foreground flex h-9 w-full min-w-0 rounded-lg px-3 py-1 text-base shadow-xs transition-all file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium disabled:cursor-not-allowed disabled:opacity-50 md:text-sm input-depth focus-border",
  {
    variants: {
      variant: {
        outline: "border border-border bg-background hover:border-border-strong focus-visible:border-primary dark:bg-input/10",
        filled: "border border-transparent bg-muted/30 hover:bg-muted/50 focus-visible:bg-background focus-visible:border-primary focus-visible:shadow-sm",
      },
      error: {
        true: "border-destructive focus-visible:border-destructive text-error placeholder:text-muted-foreground",
      }
    },
    defaultVariants: {
      variant: "outline",
    },
  }
)

export interface InputProps extends React.ComponentProps<"input">, VariantProps<typeof inputVariants> {
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
  id,
  "aria-describedby": ariaDescribedBy,
  "aria-invalid": ariaInvalid,
  ...props
}: InputProps) {
  const controlA11y = useFormControlA11y({
    id,
    error,
    errorText,
    ariaDescribedBy,
    ariaInvalid,
  })

  const inputNode = (
    <input
      id={id}
      type={type}
      data-slot="input"
      aria-describedby={controlA11y.ariaDescribedBy}
      aria-invalid={controlA11y.ariaInvalid}
      className={cn(
        inputVariants({ variant, error: controlA11y.isInvalid, className }),
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
          <FormControlError id={controlA11y.errorId}>
            {errorText}
          </FormControlError>
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

export interface PasswordInputProps
  extends Omit<InputProps, "type" | "rightIcon"> {
  showPasswordLabel: string
  hidePasswordLabel: string
}

function PasswordInput({
  className,
  showPasswordLabel,
  hidePasswordLabel,
  disabled,
  id,
  ...props
}: PasswordInputProps) {
  const [show, setShow] = React.useState(false)
  const generatedId = React.useId()
  const inputId = id ?? `password-${generatedId}`

  return (
    <Input
      id={inputId}
      type={show ? "text" : "password"}
      rightIcon={
        <button
          type="button"
          aria-label={show ? hidePasswordLabel : showPasswordLabel}
          aria-controls={inputId}
          disabled={disabled}
          onClick={() => setShow(!show)}
          className="inline-flex size-6 items-center justify-center rounded-sm transition-colors outline-hidden pointer-events-auto cursor-pointer hover:text-foreground focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
        >
          {show ? <EyeOffIcon className="size-4" /> : <EyeIcon className="size-4" />}
        </button>
      }
      className={className}
      disabled={disabled}
      {...props}
    />
  )
}

export type ColorPickerProps = Omit<
  InputProps,
  "type" | "leftIcon" | "rightIcon" | "root"
>

function ColorPicker({
  className,
  value,
  onChange,
  disabled,
  error,
  errorText,
  id,
  "aria-describedby": ariaDescribedBy,
  "aria-invalid": ariaInvalid,
  ...props
}: ColorPickerProps) {
  const [internalValue, setInternalValue] = React.useState(typeof value === 'string' ? value : "#000000")
  
  const color = typeof value === 'string' ? value : internalValue

  const controlA11y = useFormControlA11y({
    id,
    error,
    errorText,
    ariaDescribedBy,
    ariaInvalid,
  })

  return (
    <div className="flex flex-col gap-1 w-full">
      <div
        data-slot="color-picker"
        className={cn(
          "relative flex h-9 w-full items-center gap-3 rounded-lg border border-border bg-background px-3 py-1 shadow-xs transition-all hover:border-border-strong cursor-pointer input-depth focus-within:border-primary focus-within:ring-1 focus-within:ring-primary/20",
          disabled && "opacity-50 cursor-not-allowed",
          controlA11y.isInvalid && "border-destructive focus-within:border-destructive",
          className
        )}
      >
        <input
          id={id}
          type="color"
          className="absolute inset-0 z-10 size-full cursor-pointer opacity-0 disabled:cursor-not-allowed"
          value={color}
          disabled={disabled}
          aria-describedby={controlA11y.ariaDescribedBy}
          aria-invalid={controlA11y.ariaInvalid}
          onChange={(event) => {
            const newValue = event.target.value
            setInternalValue(newValue)
            onChange?.(event)
          }}
          {...props}
        />
        <div
          aria-hidden="true"
          className="size-5 rounded-full border border-border-strong/20 shadow-sm shrink-0"
          style={{ backgroundColor: color }}
        />
        <span aria-hidden="true" className="text-sm font-medium font-mono uppercase tracking-wider text-foreground/80 first-letter:uppercase">
          {color}
        </span>
      </div>
      {errorText && (
        <FormControlError id={controlA11y.errorId} className="mt-0">
          {errorText}
        </FormControlError>
      )}
    </div>
  )
}

export { Input, SearchInput, PasswordInput, ColorPicker }
