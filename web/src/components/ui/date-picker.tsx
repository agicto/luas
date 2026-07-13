/**
 * @component DatePicker
 * @category UI
 * @status Stable
 * @description Locale-aware date and time picker with explicit form semantics.
 * @usage Use for enhanced date selection. Use native Input type="date" when browser-native constraint validation is preferred.
 * @example
 * <DatePicker id="meeting-at" name="meetingAt" value={date} onChange={setDate} showTime />
 */
'use client';

import * as React from 'react';
import { useLocale } from 'next-intl';
import dynamic from 'next/dynamic';
import { CalendarIcon } from 'lucide-react';

import { useT } from '@/i18n';
import type { Locale } from '@/i18n/locales';
import { cn } from '@/utils';
import {
  parseDatePickerValue,
  serializeDatePickerValue,
} from './date-picker-value';
import { FormControlError, useFormControlA11y } from './form-control';
import { Popover, PopoverContent, PopoverTrigger } from './popover';

const Calendar = dynamic(
  () => import('./calendar').then((module) => module.Calendar),
  {
    loading: () => (
      <div aria-hidden="true" className="h-[300px] w-[260px]" />
    ),
  }
);

type DatePickerTriggerProps = Omit<
  React.ButtonHTMLAttributes<HTMLButtonElement>,
  'children' | 'defaultValue' | 'onChange' | 'type' | 'value'
>;

export interface DatePickerProps extends DatePickerTriggerProps {
  date?: Date;
  defaultValue?: Date | string;
  error?: boolean;
  errorText?: React.ReactNode;
  locale?: Locale;
  name?: string;
  onChange?: (date: Date) => void;
  placeholder?: string;
  required?: boolean;
  setDate?: (date: Date) => void;
  showTime?: boolean;
  value?: Date | string;
}

interface TimeSelectProps {
  disabled: boolean;
  label: string;
  max: number;
  onChange: (value: number) => void;
  value: number;
}

function TimeSelect({
  disabled,
  label,
  max,
  onChange,
  value,
}: TimeSelectProps) {
  return (
    <label className="grid gap-1 text-xs font-medium text-muted-foreground">
      <span>{label}</span>
      <select
        aria-label={label}
        className="h-8 rounded-md border border-border bg-background px-2 font-mono text-sm text-foreground outline-none focus:border-primary focus-ring disabled:cursor-not-allowed disabled:opacity-50"
        disabled={disabled}
        value={value}
        onChange={(event) => onChange(Number(event.target.value))}
      >
        {Array.from({ length: max + 1 }, (_, option) => (
          <option key={option} value={option}>
            {String(option).padStart(2, '0')}
          </option>
        ))}
      </select>
    </label>
  );
}

export function DatePicker({
  'aria-describedby': ariaDescribedBy,
  'aria-invalid': ariaInvalid,
  className,
  date: legacyDate,
  defaultValue,
  disabled = false,
  error,
  errorText,
  form,
  id,
  locale: localeOverride,
  name,
  onChange,
  placeholder,
  required = false,
  setDate: legacySetDate,
  showTime = false,
  value,
  ...triggerProps
}: DatePickerProps) {
  const t = useT('common');
  const requestLocale = useLocale() as Locale;
  const locale = localeOverride ?? requestLocale;
  const generatedId = React.useId();
  const triggerId = id ?? `date-picker-${generatedId}`;
  const dialogId = `${triggerId}-dialog`;
  const isControlled = legacyDate !== undefined || value !== undefined;
  const controlledDate = parseDatePickerValue(legacyDate ?? value);
  const [internalDate, setInternalDate] = React.useState(() =>
    parseDatePickerValue(defaultValue)
  );
  const [open, setOpen] = React.useState(false);
  const selectedDate = isControlled ? controlledDate : internalDate;
  const controlA11y = useFormControlA11y({
    id: triggerId,
    error,
    errorText,
    ariaDescribedBy,
    ariaInvalid,
  });
  const formatter = React.useMemo(
    () =>
      new Intl.DateTimeFormat(locale, {
        dateStyle: 'medium',
        ...(showTime ? { timeStyle: 'medium' as const } : {}),
      }),
    [locale, showTime]
  );

  const commitDate = (nextDate: Date) => {
    if (!isControlled) {
      setInternalDate(nextDate);
    }
    legacySetDate?.(nextDate);
    onChange?.(nextDate);
  };

  const handleDateSelect = (nextDate: Date) => {
    const committedDate = new Date(nextDate);
    if (showTime && selectedDate) {
      committedDate.setHours(
        selectedDate.getHours(),
        selectedDate.getMinutes(),
        selectedDate.getSeconds(),
        selectedDate.getMilliseconds()
      );
    }
    commitDate(committedDate);
  };

  const handleTimeChange = (
    unit: 'hours' | 'minutes' | 'seconds',
    nextValue: number
  ) => {
    if (!selectedDate) {
      return;
    }
    const nextDate = new Date(selectedDate);
    if (unit === 'hours') nextDate.setHours(nextValue);
    if (unit === 'minutes') nextDate.setMinutes(nextValue);
    if (unit === 'seconds') nextDate.setSeconds(nextValue);
    commitDate(nextDate);
  };

  return (
    <div className="grid w-full gap-1" data-slot="date-picker">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button
            {...triggerProps}
            id={triggerId}
            type="button"
            role="combobox"
            aria-haspopup="dialog"
            aria-controls={dialogId}
            aria-expanded={open}
            form={form}
            disabled={disabled}
            aria-describedby={controlA11y.ariaDescribedBy}
            aria-invalid={controlA11y.ariaInvalid}
            aria-required={required || undefined}
            className={cn(
              'flex h-9 w-full items-center justify-between rounded-lg border border-border bg-background px-3 py-1 text-left text-sm font-normal shadow-xs transition-all input-depth hover:border-border-strong focus:border-primary focus:outline-hidden',
              !selectedDate && 'text-muted-foreground',
              controlA11y.isInvalid &&
                'border-destructive text-destructive focus:border-destructive',
              className
            )}
          >
            <span className="truncate">
              {selectedDate ? formatter.format(selectedDate) : placeholder ?? t('datePlaceholder')}
            </span>
            <CalendarIcon
              aria-hidden="true"
              className={cn(
                'ml-2 size-4 shrink-0 opacity-50',
                controlA11y.isInvalid && 'text-destructive opacity-100'
              )}
            />
          </button>
        </PopoverTrigger>

        <PopoverContent
          id={dialogId}
          role="dialog"
          aria-label={t('chooseDate')}
          className="w-auto overflow-hidden p-0"
          align="start"
        >
          <div className="flex flex-col bg-popover/30 backdrop-blur-2xl sm:flex-row">
            <Calendar
              autoFocus
              locale={locale}
              selected={selectedDate}
              showFooter={false}
              onSelect={handleDateSelect}
            />

            {showTime && (
              <fieldset className="grid content-start gap-3 border-t border-border/40 p-3 sm:border-l sm:border-t-0">
                <legend className="sr-only">{t('time')}</legend>
                <TimeSelect
                  disabled={disabled || !selectedDate}
                  label={t('hour')}
                  max={23}
                  value={selectedDate?.getHours() ?? 0}
                  onChange={(nextValue) =>
                    handleTimeChange('hours', nextValue)
                  }
                />
                <TimeSelect
                  disabled={disabled || !selectedDate}
                  label={t('minute')}
                  max={59}
                  value={selectedDate?.getMinutes() ?? 0}
                  onChange={(nextValue) =>
                    handleTimeChange('minutes', nextValue)
                  }
                />
                <TimeSelect
                  disabled={disabled || !selectedDate}
                  label={t('second')}
                  max={59}
                  value={selectedDate?.getSeconds() ?? 0}
                  onChange={(nextValue) =>
                    handleTimeChange('seconds', nextValue)
                  }
                />
              </fieldset>
            )}
          </div>

          <div className="flex items-center justify-between border-t border-border bg-muted/10 p-2">
            <button
              type="button"
              className="rounded-md px-3 py-1.5 text-xs font-bold text-foreground/90 transition-colors hover:bg-primary/5 hover:text-primary focus-ring"
              onClick={() => commitDate(new Date())}
            >
              {t('now')}
            </button>
            <button
              type="button"
              className="rounded-lg bg-primary px-6 py-1.5 text-xs font-bold text-primary-foreground shadow-button-primary transition-all hover:brightness-110 focus-ring"
              onClick={() => setOpen(false)}
            >
              {t('confirm')}
            </button>
          </div>
        </PopoverContent>
      </Popover>

      {name && (
        <input
          type="hidden"
          name={name}
          form={form}
          disabled={disabled}
          value={serializeDatePickerValue(selectedDate, showTime)}
        />
      )}

      {errorText && (
        <FormControlError id={controlA11y.errorId} className="mt-0">
          {errorText}
        </FormControlError>
      )}
    </div>
  );
}
