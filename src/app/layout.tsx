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
import { publicEnv } from "@/config/env";
import "./globals.css";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: {
    default: "Llamacto Web Scaffold",
    template: "%s | Llamacto",
  },
  description: "Modern web application scaffold built with Next.js, TypeScript, and Tailwind CSS",
  keywords: ["Next.js", "React", "TypeScript", "Tailwind CSS", "Scaffold"],
  authors: [{ name: "Llamacto Team" }],
  creator: "Llamacto Team",
  metadataBase: new URL(publicEnv.NEXT_PUBLIC_APP_URL),

  openGraph: {
    type: "website",
    locale: "en_US",
    siteName: "Llamacto Web Scaffold",
  },
  twitter: {
    card: "summary_large_image",
  },
  robots: {
    index: true,
    follow: true,
  },
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
