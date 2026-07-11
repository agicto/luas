'use client';

import { useEffect } from 'react';

import { useT } from '@/i18n';

interface AppErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function AppError({ error, reset }: AppErrorProps) {
  const t = useT();

  useEffect(() => {
    console.error('[Luas] Uncaught route error', error, {
      digest: error.digest,
    });
  }, [error]);

  return (
    <main
      className="flex min-h-[50svh] items-center justify-center px-6 py-16 text-center"
      role="alert"
      aria-live="assertive"
    >
      <div className="max-w-md">
        <h1 className="text-2xl font-semibold text-text-main">
          {t.errors('serverError')}
        </h1>
        <p className="mt-3 text-sm leading-6 text-text-subtle">
          {t.common('retryLater')}
        </p>
        <button
          type="button"
          onClick={reset}
          className="mt-6 inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        >
          {t.common('retry')}
        </button>
      </div>
    </main>
  );
}
