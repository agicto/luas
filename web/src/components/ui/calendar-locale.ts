import type { DayPickerLocale } from 'react-day-picker';
import { enUS } from 'react-day-picker/locale/en-US';
import { zhCN } from 'react-day-picker/locale/zh-CN';

import type { Locale } from '@/i18n/locales';

const calendarLocales = {
  'en-US': enUS,
  'zh-Hans': zhCN,
} satisfies Record<Locale, DayPickerLocale>;

export function getCalendarLocale(locale: Locale): DayPickerLocale {
  return calendarLocales[locale];
}
