import Link from "next/link";
import { Activity, ArrowUpRight, BookOpen, Code2, Rocket } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ROUTES } from "@/constants/routes";
import { getSessionUser } from "@/features/auth/server/session";

import { GreetingClock } from "./_components/greeting-clock";

/**
 * Console dashboard (RSC).
 *
 * Default landing for authenticated users. Kept deliberately minimal —
 * this is a scaffold example, not a product. Replace with your real
 * domain UI.
 *
 * Pattern: page is a Server Component; the only client island is
 * `GreetingClock` which re-renders the wall-clock every minute.
 */
export default async function ConsoleHomePage() {
  const user = await getSessionUser();

  const quickLinks = [
    {
      title: "API docs",
      description: "Read the Luas Go backend OpenAPI spec.",
      href: ROUTES.CONSOLE.SETTINGS,
      icon: BookOpen,
    },
    {
      title: "Styleguide",
      description: "Browse the design system and component gallery.",
      href: ROUTES.DEVTOOLS.STYLEGUIDE,
      icon: Code2,
    },
    {
      title: "i18n test",
      description: "Inspect the next-intl translation tree.",
      href: ROUTES.DEVTOOLS.I18N_TEST,
      icon: Activity,
    },
  ] as const;

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8 md:px-8 md:py-12">
      <header className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <GreetingClock name={user?.name ?? user?.email ?? "there"} />
          <p className="mt-1 text-sm text-text-muted">
            Welcome to your Luas console. This page is an example — replace it with your dashboard.
          </p>
        </div>
        <Button asChild>
          <Link href={ROUTES.CONSOLE.SETTINGS}>
            Open settings
            <ArrowUpRight className="ml-1.5 h-4 w-4" />
          </Link>
        </Button>
      </header>

      <section className="grid gap-4 md:grid-cols-3">
        {quickLinks.map(({ title, description, href, icon: Icon }) => (
          <Card key={href} className="transition-colors hover:border-primary/40">
            <CardHeader className="flex flex-row items-start justify-between gap-3 space-y-0">
              <div>
                <CardTitle className="text-base">{title}</CardTitle>
                <CardDescription className="mt-1">{description}</CardDescription>
              </div>
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <Icon className="h-4.5 w-4.5" />
              </div>
            </CardHeader>
            <CardContent>
              <Button asChild variant="ghost" size="sm" className="-ml-2">
                <Link href={href}>
                  Open
                  <ArrowUpRight className="ml-1 h-3.5 w-3.5" />
                </Link>
              </Button>
            </CardContent>
          </Card>
        ))}
      </section>

      <section className="mt-10 rounded-xl border bg-bg-surface p-6">
        <div className="flex items-start gap-4">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Rocket className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-base font-semibold">Next steps</h2>
            <ul className="mt-2 space-y-1 text-sm text-text-muted">
              <li>
                Wire <code className="font-mono text-xs">NEXT_PUBLIC_API_URL</code> to your Luas
                Go backend (or your own API).
              </li>
              <li>
                Replace this page (
                <code className="font-mono text-xs">
                  src/app/(protected)/(console)/console/page.tsx
                </code>
                ) with your real dashboard.
              </li>
              <li>
                Drop new features into <code className="font-mono text-xs">src/features/</code>{" "}
                and expose them through new routes under{" "}
                <code className="font-mono text-xs">(protected)/(console)/</code>.
              </li>
            </ul>
          </div>
        </div>
      </section>
    </div>
  );
}
