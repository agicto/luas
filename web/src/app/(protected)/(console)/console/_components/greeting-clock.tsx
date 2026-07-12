'use client';

import { useEffect, useState } from 'react';
import { useLocale } from 'next-intl';
import { useT } from '@/i18n';

/**
 * Small client island for the console greeting + wall-clock.
 * Keeps the parent page a pure Server Component.
 */
export function GreetingClock({ name }: { name: string }) {
  const [now, setNow] = useState<Date | null>(null);
  const locale = useLocale();
  const t = useT('console.greeting');

  useEffect(() => {
    const update = () => setNow(new Date());
    const firstTick = window.setTimeout(update, 0);
    const interval = window.setInterval(update, 60_000);
    return () => {
      window.clearTimeout(firstTick);
      window.clearInterval(interval);
    };
  }, []);

  const greetingKey = pickGreeting(now);
  const time = now?.toLocaleTimeString(locale, {
    hour: '2-digit',
    minute: '2-digit',
  });

  return (
    <h1 className="text-2xl font-semibold tracking-tight md:text-3xl">
      {t(greetingKey, { name })}
      {time ? (
        <>
          {' '}
          <span className="ml-3 align-middle text-base font-normal text-text-muted">{time}</span>
        </>
      ) : null}
    </h1>
  );
}

type GreetingKey = 'hello' | 'late' | 'morning' | 'afternoon' | 'evening';

function pickGreeting(now: Date | null): GreetingKey {
  if (!now) return 'hello';
  const h = now.getHours();
  if (h < 5) return 'late';
  if (h < 12) return 'morning';
  if (h < 18) return 'afternoon';
  return 'evening';
}
