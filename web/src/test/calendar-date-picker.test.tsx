import type { ComponentProps, ReactNode } from 'react';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import {
  NextIntlClientProvider,
  type AbstractIntlMessages,
} from 'next-intl';
import { describe, expect, expectTypeOf, it, vi } from 'vitest';

import { Calendar } from '@/components/ui/calendar';
import { DatePicker } from '@/components/ui/date-picker';
import {
  parseDatePickerValue,
  serializeDatePickerValue,
} from '@/components/ui/date-picker-value';
import { Label } from '@/components/ui/label';
import type { Locale } from '@/i18n';
import { messages as enMessages } from '@/i18n/modules';
import zhCommon from '@/i18n/modules/common/zh-Hans';

function renderWithLocale(children: ReactNode, locale: Locale = 'en-US') {
  const localeMessages = (
    locale === 'en-US' ? enMessages : { common: zhCommon }
  ) as unknown as AbstractIntlMessages;

  return render(
    <NextIntlClientProvider
      locale={locale}
      messages={localeMessages}
    >
      {children}
    </NextIntlClientProvider>
  );
}

describe('calendar and date picker contracts', () => {
  it('uses locale week order and translated calendar navigation names', () => {
    const { container, rerender } = renderWithLocale(
      <Calendar
        selected={new Date(2026, 6, 13)}
        showFooter={false}
        today={new Date(2026, 6, 10)}
      />
    );
    const firstWeekday = () => container.querySelector('thead th');

    expect(screen.getByRole('grid')).toHaveAccessibleName('July 2026');
    expect(firstWeekday()).toHaveAttribute('aria-label', 'Sunday');
    expect(
      screen.getByRole('button', { name: 'Go to the Previous Month' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Go to the Next Month' })
    ).toBeInTheDocument();

    rerender(
      <NextIntlClientProvider
        locale="zh-Hans"
        messages={{ common: zhCommon } as unknown as AbstractIntlMessages}
      >
        <Calendar
          selected={new Date(2026, 6, 13)}
          showFooter={false}
          today={new Date(2026, 6, 10)}
        />
      </NextIntlClientProvider>
    );

    expect(screen.getByRole('grid')).toHaveAccessibleName('2026年7月');
    expect(firstWeekday()).toHaveAttribute('aria-label', '星期一');
    expect(
      screen.getByRole('button', { name: '前往上个月' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '前往下个月' })
    ).toBeInTheDocument();
  });

  it('moves day focus with calendar keyboard navigation', async () => {
    renderWithLocale(
      <Calendar
        selected={new Date(2026, 6, 13)}
        showFooter={false}
        today={new Date(2026, 6, 10)}
      />
    );

    const day13 = document.querySelector<HTMLButtonElement>(
      '[data-day="2026-07-13"] button'
    );
    const day14 = document.querySelector<HTMLButtonElement>(
      '[data-day="2026-07-14"] button'
    );

    expect(day13).not.toBeNull();
    expect(day14).not.toBeNull();

    await act(async () => {
      day13?.focus();
    });
    await act(async () => {
      fireEvent.keyDown(day13 as HTMLButtonElement, { key: 'ArrowRight' });
    });

    await waitFor(() => expect(day14).toHaveFocus());
  });

  it('preserves form semantics and serializes a local date value', () => {
    type DatePickerProps = ComponentProps<typeof DatePicker>;

    expectTypeOf<DatePickerProps>().toMatchTypeOf<{
      id?: string;
      name?: string;
      required?: boolean;
      disabled?: boolean;
      'aria-describedby'?: string;
    }>();

    const pickerProps = {
      id: 'start-date',
      name: 'startDate',
      required: true,
      value: new Date(2026, 6, 13),
    } as unknown as DatePickerProps;

    const { container } = renderWithLocale(
      <>
        <Label htmlFor="start-date">Start date</Label>
        <DatePicker {...pickerProps} />
      </>
    );

    const trigger = screen.getByLabelText('Start date');
    const formValue = container.querySelector<HTMLInputElement>(
      'input[type="hidden"][name="startDate"]'
    );

    expect(trigger).toHaveRole('combobox');
    expect(trigger).toHaveAttribute('aria-haspopup', 'dialog');
    expect(trigger).toHaveAttribute('aria-expanded', 'false');
    expect(trigger).toHaveAttribute('aria-controls', 'start-date-dialog');
    expect(trigger).toHaveAttribute('aria-required', 'true');
    expect(trigger).toHaveTextContent('Jul 13, 2026');
    expect(formValue).toHaveValue('2026-07-13');
  });

  it('uses the configured today value without exposing it for mutation', () => {
    const today = new Date(2030, 0, 2, 12);
    const onSelect = vi.fn();

    renderWithLocale(<Calendar today={today} onSelect={onSelect} />);
    fireEvent.click(screen.getByRole('button', { name: 'Today' }));

    const emittedDate = onSelect.mock.calls[0]?.[0] as Date | undefined;
    expect(emittedDate).toEqual(today);
    expect(emittedDate).not.toBe(today);
  });

  it('associates picker errors without replacing existing descriptions', () => {
    type DatePickerProps = ComponentProps<typeof DatePicker>;
    const pickerProps = {
      id: 'due-date',
      'aria-label': 'Due date',
      'aria-describedby': 'due-date-hint',
      errorText: 'Choose a due date',
    } as unknown as DatePickerProps;

    renderWithLocale(
      <>
        <DatePicker {...pickerProps} />
        <p id="due-date-hint">Dates use your current time zone.</p>
      </>
    );

    const trigger = screen.getByLabelText('Due date');
    const error = screen.getByText('Choose a due date');

    expect(trigger).toHaveAttribute('aria-invalid', 'true');
    expect(trigger).toHaveAttribute(
      'aria-describedby',
      `due-date-hint ${error.id}`
    );
    expect(error).toHaveAttribute('aria-live', 'polite');
  });

  it('parses and serializes local date values without UTC date shifts', () => {
    const date = parseDatePickerValue('2026-07-13T09:05:04');

    expect(date).toBeDefined();
    expect(date?.getFullYear()).toBe(2026);
    expect(date?.getMonth()).toBe(6);
    expect(date?.getDate()).toBe(13);
    expect(date?.getHours()).toBe(9);
    expect(date?.getMinutes()).toBe(5);
    expect(date?.getSeconds()).toBe(4);
    expect(parseDatePickerValue('2026-02-30')).toBeUndefined();
    expect(serializeDatePickerValue(date, false)).toBe('2026-07-13');
    expect(serializeDatePickerValue(date, true)).toBe(
      '2026-07-13T09:05:04'
    );
  });

  it('exposes a named dialog and emits time changes without mutating input', async () => {
    const originalDate = new Date(2026, 6, 13, 9, 5, 4);
    const onChange = vi.fn();

    renderWithLocale(
      <DatePicker
        aria-label="Meeting time"
        value={originalDate}
        onChange={onChange}
        showTime
      />
    );

    fireEvent.click(screen.getByLabelText('Meeting time'));

    const dialog = await screen.findByRole('dialog', { name: 'Choose date' });
    expect(dialog).toHaveAttribute(
      'id',
      screen.getByLabelText('Meeting time').getAttribute('aria-controls')
    );
    expect(screen.getByLabelText('Meeting time')).toHaveAttribute(
      'aria-expanded',
      'true'
    );
    await waitFor(() =>
      expect(
        document.querySelector('[data-selected] button')
      ).toHaveFocus()
    );
    fireEvent.change(screen.getByRole('combobox', { name: 'Hour' }), {
      target: { value: '10' },
    });

    const emittedDate = onChange.mock.calls[0]?.[0] as Date | undefined;
    expect(emittedDate?.getHours()).toBe(10);
    expect(emittedDate?.getMinutes()).toBe(5);
    expect(originalDate.getHours()).toBe(9);

    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }));
    await waitFor(() =>
      expect(
        screen.queryByRole('dialog', { name: 'Choose date' })
      ).not.toBeInTheDocument()
    );
  });
});
