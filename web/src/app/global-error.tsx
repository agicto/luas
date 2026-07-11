'use client';

import { useEffect } from 'react';

import './globals.css';

interface GlobalErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

export default function GlobalError({ error, reset }: GlobalErrorProps) {
  useEffect(() => {
    console.error('[Luas] Uncaught root error', error, {
      digest: error.digest,
    });
  }, [error]);

  return (
    <html lang="en">
      <head>
        <title>Luas | Application error</title>
      </head>
      <body className="min-h-screen bg-bg-canvas text-text-main antialiased">
        <main
          className="flex min-h-screen items-center justify-center px-6 py-16 text-center"
          role="alert"
          aria-live="assertive"
        >
          <div className="max-w-md">
            <h1 className="text-2xl font-semibold">Application error</h1>
            <p className="mt-3 text-sm leading-6 text-text-subtle">
              The application could not be loaded. Retry the request.
            </p>
            <button
              type="button"
              onClick={reset}
              className="mt-6 inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            >
              Try again
            </button>
          </div>
        </main>
      </body>
    </html>
  );
}
