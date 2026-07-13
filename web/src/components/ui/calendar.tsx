/**
 * @component Calendar
 * @category UI
 * @status Stable
 * @description Locale-aware single-date calendar backed by React DayPicker.
 * @usage Use for date selection inside forms and filters. Date grid semantics, focus, and keyboard navigation are owned by React DayPicker.
 * @example
 * <Calendar selected={date} onSelect={setDate} />
 */
'use client';

import * as React from 'react';
import { useLocale } from 'next-intl';
import {
  DayFlag,
  DayPicker,
  SelectionState,
  UI,
  type ChevronProps,
  type DayPickerProps,
} from 'react-day-picker';
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
} from 'lucide-react';

import { useT } from '@/i18n';
import type { Locale } from '@/i18n/locales';
import { cn } from '@/utils';
import { getCalendarLocale } from './calendar-locale';

type DayPickerBaseProps = Omit<
  DayPickerProps,
  'footer' | 'locale' | 'mode' | 'onSelect' | 'required' | 'selected'
>;

export interface CalendarProps extends DayPickerBaseProps {
  locale?: Locale;
  onSelect?: (date: Date) => void;
  onToday?: () => void;
  selected?: Date;
  showFooter?: boolean;
}

function CalendarChevron({
  className,
  orientation = 'left',
  size = 16,
  style,
}: ChevronProps) {
  const Icon = {
    down: ChevronDown,
    left: ChevronLeft,
    right: ChevronRight,
    up: ChevronUp,
  }[orientation];

  return (
    <Icon
      aria-hidden="true"
      className={cn('size-4', className)}
      size={size}
      style={style}
    />
  );
}

export function Calendar({
  className,
  classNames,
  components,
  defaultMonth,
  locale: localeOverride,
  month: controlledMonth,
  onMonthChange,
  onSelect,
  onToday,
  selected,
  showFooter = true,
  startMonth,
  endMonth,
  today: todayOverride,
  ...props
}: CalendarProps) {
  const t = useT('common');
  const requestLocale = useLocale() as Locale;
  const locale = localeOverride ?? requestLocale;
  const today = React.useMemo(
    () => todayOverride ?? new Date(),
    [todayOverride]
  );
  const [internalMonth, setInternalMonth] = React.useState(
    controlledMonth ?? selected ?? defaultMonth ?? today
  );
  const visibleMonth = controlledMonth ?? internalMonth;
  const dateFormatter = React.useMemo(
    () => new Intl.DateTimeFormat(locale, { dateStyle: 'full' }),
    [locale]
  );

  React.useEffect(() => {
    if (selected && !controlledMonth) {
      setInternalMonth((currentMonth) =>
        currentMonth.getFullYear() === selected.getFullYear() &&
        currentMonth.getMonth() === selected.getMonth()
          ? currentMonth
          : selected
      );
    }
  }, [controlledMonth, selected]);

  const handleMonthChange = (nextMonth: Date) => {
    if (!controlledMonth) {
      setInternalMonth(nextMonth);
    }
    onMonthChange?.(nextMonth);
  };

  const handleToday = () => {
    const currentDate = new Date(today);
    handleMonthChange(currentDate);
    onSelect?.(currentDate);
    onToday?.();
  };

  return (
    <div className={cn('w-fit', className)} data-slot="calendar">
      <DayPicker
        {...props}
        mode="single"
        required
        selected={selected}
        onSelect={onSelect}
        month={visibleMonth}
        onMonthChange={handleMonthChange}
        startMonth={startMonth ?? new Date(today.getFullYear() - 100, 0, 1)}
        endMonth={endMonth ?? new Date(today.getFullYear() + 10, 11, 1)}
        locale={getCalendarLocale(locale)}
        lang={locale}
        today={today}
        captionLayout="dropdown"
        navLayout="after"
        fixedWeeks
        showOutsideDays
        footer={
          selected
            ? t('selectedDate', { date: dateFormatter.format(selected) })
            : undefined
        }
        components={{ Chevron: CalendarChevron, ...components }}
        classNames={{
          [UI.Root]: 'relative p-3',
          [UI.Months]: 'relative flex flex-col gap-4',
          [UI.Month]: 'flex w-full flex-col gap-3',
          [UI.MonthCaption]: 'flex h-8 items-center justify-center px-16',
          [UI.Dropdowns]: 'flex items-center justify-center gap-1',
          [UI.DropdownRoot]:
            'relative inline-flex h-8 items-center rounded-md border border-border bg-background shadow-xs focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/20',
          [UI.Dropdown]:
            'absolute inset-0 z-10 h-full w-full cursor-pointer opacity-0',
          [UI.CaptionLabel]:
            'inline-flex items-center gap-1 px-2 text-sm font-semibold',
          [UI.Nav]:
            'pointer-events-none absolute inset-x-3 top-3 z-10 flex items-center justify-between',
          [UI.PreviousMonthButton]:
            'pointer-events-auto inline-flex size-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-ring disabled:pointer-events-none disabled:opacity-50',
          [UI.NextMonthButton]:
            'pointer-events-auto inline-flex size-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted hover:text-foreground focus-ring disabled:pointer-events-none disabled:opacity-50',
          [UI.Chevron]: 'fill-current text-current',
          [UI.MonthGrid]: 'w-full border-collapse',
          [UI.Weekdays]: 'flex',
          [UI.Weekday]:
            'w-8 text-center text-[0.7rem] font-medium text-muted-foreground',
          [UI.Weeks]: 'block',
          [UI.Week]: 'mt-1 flex w-full',
          [UI.Day]: 'relative size-8 p-0 text-center text-sm',
          [UI.DayButton]:
            'inline-flex size-8 items-center justify-center rounded-md font-medium transition-colors hover:bg-muted focus-ring disabled:pointer-events-none',
          [SelectionState.selected]:
            'rounded-md bg-primary text-primary-foreground shadow-button-primary [&>button]:font-semibold [&>button]:hover:bg-primary',
          [DayFlag.today]:
            'rounded-md bg-accent text-accent-foreground after:absolute after:bottom-1 after:left-1/2 after:size-1 after:-translate-x-1/2 after:rounded-full after:bg-primary',
          [DayFlag.outside]: 'text-muted-foreground opacity-45',
          [DayFlag.disabled]: 'text-muted-foreground opacity-40',
          [DayFlag.hidden]: 'invisible',
          [UI.Footer]: 'sr-only',
          ...classNames,
        }}
      />

      {showFooter && (
        <div className="border-t border-border px-3 py-2 text-xs">
          <button
            type="button"
            className="rounded-md px-2 py-1 font-semibold text-muted-foreground transition-colors hover:bg-primary/5 hover:text-primary focus-ring"
            onClick={handleToday}
          >
            {t('today')}
          </button>
        </div>
      )}
    </div>
  );
}
