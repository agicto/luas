'use client';

import { useLocale } from '@/hooks/use-locale';
import { locales, localeNames, isLocaleSwitcherEnabled, type Locale } from '@/i18n';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Globe } from 'lucide-react';

export function LocaleSwitcher() {
  const { locale, setLocale, isPending } = useLocale();

  // Don't render if locale switcher is disabled via environment variable
  if (!isLocaleSwitcherEnabled) {
    return null;
  }

  return (
    <Select
      value={locale}
      onValueChange={(value) => setLocale(value as Locale)}
      disabled={isPending}
    >
      <SelectTrigger className="w-[140px]">
        <Globe className="mr-2 h-4 w-4" />
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {locales.map((loc) => (
          <SelectItem key={loc} value={loc}>
            {localeNames[loc]}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
