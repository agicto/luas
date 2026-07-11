import type { Metadata } from "next";
import { NextIntlClientProvider } from "next-intl";
import { getLocale, getMessages } from "next-intl/server";
import { cn } from "@/utils";
import { ThemeProvider } from "@/providers/theme-provider";
import { Analytics } from "@/components/analytics";
import { env } from "@/config/env";
import { CLIENT_MESSAGE_NAMESPACES } from "@/i18n/client-message-namespaces";
import { selectMessageNamespaces } from "@/i18n/message-selection";
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
  const clientMessages = selectMessageNamespaces(
    messages,
    CLIENT_MESSAGE_NAMESPACES.global
  );

  return (
    <html lang={locale} suppressHydrationWarning>
      <body className={cn("min-h-screen font-sans antialiased")}>
        <NextIntlClientProvider messages={clientMessages}>
          <ThemeProvider
            attribute="class"
            defaultTheme="system"
            enableSystem
            disableTransitionOnChange
          >
            {children}
          </ThemeProvider>
        </NextIntlClientProvider>
        <Analytics />
      </body>
    </html>
  );
}
