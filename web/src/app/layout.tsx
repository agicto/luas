import type { Metadata } from "next";
import { NextIntlClientProvider } from "next-intl";
import { getLocale, getMessages } from "next-intl/server";
import { cn } from "@/utils";
import { ThemeProvider } from "@/providers/theme-provider";
import { ErrorBoundary } from "@/components/error-boundary";
import { Toaster } from "@/components/ui/sonner";
import { Analytics } from "@/components/analytics";
import { env } from "@/config/env";
import "@/config/server-env";
import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: "Luas",
    template: "%s | Luas",
  },
  description: "Modern web application scaffold built with Next.js, TypeScript, and Tailwind CSS",
  metadataBase: new URL(env.NEXT_PUBLIC_APP_URL),
};

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const locale = await getLocale();
  const messages = await getMessages();

  return (
    <html lang={locale} suppressHydrationWarning>
      <body className={cn("min-h-screen font-sans antialiased")}>
        <ErrorBoundary>
          <NextIntlClientProvider messages={messages}>
            <ThemeProvider
              attribute="class"
              defaultTheme="system"
              enableSystem
              disableTransitionOnChange
            >
              {children}
              <Toaster richColors position="top-right" />
            </ThemeProvider>
          </NextIntlClientProvider>
        </ErrorBoundary>
        <Analytics />
      </body>
    </html>
  );
}
