'use client';

import { useEffect, useState } from 'react';

/**
 * Small client island for the dashboard greeting + wall-clock.
 * Keeps the parent page a pure Server Component.
 */
export function GreetingClock({ name }: { name: string }) {
  const [now, setNow] = useState<Date | null>(null);

  useEffect(() => {
    const update = () => setNow(new Date());
    const firstTick = window.setTimeout(update, 0);
    const interval = window.setInterval(update, 60_000);
    return () => {
      window.clearTimeout(firstTick);
      window.clearInterval(interval);
    };
  }, []);

  const greeting = pickGreeting(now);
  const time = now?.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });

  return (
    <h1 className="text-2xl font-semibold tracking-tight md:text-3xl">
      {greeting}, {name}
      {time ? <span className="ml-3 align-middle text-base font-normal text-text-muted">{time}</span> : null}
    </h1>
  );
}

function pickGreeting(now: Date | null): string {
  if (!now) return 'Hello';
  const h = now.getHours();
  if (h < 5) return 'Up late';
  if (h < 12) return 'Good morning';
  if (h < 18) return 'Good afternoon';
  return 'Good evening';
}
