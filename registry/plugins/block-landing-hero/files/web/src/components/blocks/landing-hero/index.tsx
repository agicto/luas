/**
 * @block LandingHero
 * @description A landing page hero: eyebrow badge, page heading, supporting copy, primary and
 * secondary calls to action, optional media, and a feature grid. All copy and actions arrive
 * through props; the block keeps no state and fetches nothing, so it is safe in Server Components.
 */
import * as React from 'react';

import { Badge } from '@/components/ui/badge';
import { cn } from '@/utils';

export interface LandingFeature {
  id: string;
  title: React.ReactNode;
  description: React.ReactNode;
  /** Decorative icon, for example a lucide-react icon. */
  icon?: React.ReactNode;
}

export interface LandingHeroProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  /** Short label above the heading, for example a release announcement. */
  eyebrow?: React.ReactNode;
  /** Main call to action, for example <Button asChild size="lg"><Link …/></Button>. */
  primaryAction?: React.ReactNode;
  secondaryAction?: React.ReactNode;
  /** Product image or illustration beside the copy on wide screens. */
  media?: React.ReactNode;
  features?: LandingFeature[];
  /** Heading of the feature grid; visually hidden when featuresHeadingHidden is set. */
  featuresHeading?: React.ReactNode;
  featuresHeadingHidden?: boolean;
  className?: string;
}

export function LandingHero({
  title,
  description,
  eyebrow,
  primaryAction,
  secondaryAction,
  media,
  features,
  featuresHeading,
  featuresHeadingHidden = false,
  className,
}: LandingHeroProps) {
  const idPrefix = React.useId();
  const headingId = `${idPrefix}-heading`;
  const featuresHeadingId = `${idPrefix}-features`;

  return (
    <div className={cn('flex flex-col gap-16 py-12 sm:py-16 lg:py-24', className)}>
      <section
        aria-labelledby={headingId}
        className={cn(
          'grid items-center gap-10',
          media ? 'lg:grid-cols-2' : 'mx-auto max-w-3xl text-center'
        )}
      >
        <div className={cn('flex flex-col gap-6', media ? 'items-start' : 'items-center')}>
          {eyebrow ? (
            <Badge variant="outline" className="rounded-full px-3 py-1">
              {eyebrow}
            </Badge>
          ) : null}
          <h1
            id={headingId}
            className="text-4xl font-semibold tracking-tight text-balance text-foreground sm:text-5xl lg:text-6xl"
          >
            {title}
          </h1>
          {description ? (
            <p className="max-w-2xl text-lg text-pretty text-muted-foreground">{description}</p>
          ) : null}
          {primaryAction || secondaryAction ? (
            <div
              className={cn(
                'flex w-full flex-col gap-3 sm:w-auto sm:flex-row',
                !media && 'sm:justify-center'
              )}
            >
              {primaryAction}
              {secondaryAction}
            </div>
          ) : null}
        </div>
        {media ? (
          <div className="min-w-0 overflow-hidden rounded-xl border bg-muted">{media}</div>
        ) : null}
      </section>

      {features && features.length > 0 ? (
        <section
          aria-labelledby={featuresHeading ? featuresHeadingId : undefined}
          className="flex flex-col gap-8"
        >
          {featuresHeading ? (
            <h2
              id={featuresHeadingId}
              className={cn(
                'text-center text-2xl font-semibold tracking-tight text-foreground',
                featuresHeadingHidden && 'sr-only'
              )}
            >
              {featuresHeading}
            </h2>
          ) : null}
          <ul role="list" className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {features.map(feature => (
              <li key={feature.id} className="flex flex-col gap-3 rounded-xl border bg-card p-6">
                {feature.icon ? (
                  <span
                    aria-hidden="true"
                    className="grid size-10 place-items-center rounded-lg bg-brand-subtle text-brand-strong [&>svg]:size-5"
                  >
                    {feature.icon}
                  </span>
                ) : null}
                <h3 className="text-base font-semibold text-card-foreground">{feature.title}</h3>
                <p className="text-sm text-muted-foreground">{feature.description}</p>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  );
}
