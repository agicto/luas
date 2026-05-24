import type { Metadata } from "next";
import { Inter } from "next/font/google";
import { NextIntlClientProvider } from "next-intl";
import { getLocale, getMessages } from "next-intl/server";
import { cn } from "@/utils";
import { ThemeProvider } from "@/providers/theme-provider";
import { QueryProvider } from "@/providers/query-provider";
import { AuthProvider } from "@/providers/auth-provider";
import { ErrorBoundary } from "@/components/error-boundary";
import { Toaster } from "@/components/ui/sonner";
import { Analytics } from "@/components/analytics";
import { env } from "@/config/env";
import "./globals.css";

const inter = Inter({ subsets: ["latin"] });

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
      <body className={cn(inter.className, "min-h-screen antialiased")}>
        <ErrorBoundary>
          <NextIntlClientProvider messages={messages}>
            <ThemeProvider
              attribute="class"
              defaultTheme="system"
              enableSystem
              disableTransitionOnChange
            >
              <QueryProvider>
                <AuthProvider>
                  {children}
                </AuthProvider>
              </QueryProvider>
              <Toaster richColors position="top-right" />
            </ThemeProvider>
          </NextIntlClientProvider>
        </ErrorBoundary>
        <Analytics />
      </body>
    </html>
  );
}
