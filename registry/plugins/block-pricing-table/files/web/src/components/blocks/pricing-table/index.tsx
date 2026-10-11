/**
 * @block PricingTable
 * @description Plan cards with prices, feature lists, call-to-action slots, and one optionally
 * highlighted plan. All copy and prices arrive through props, already formatted for the locale;
 * the block keeps no state and fetches nothing, so it is safe in Server Components.
 */
import * as React from 'react';
import { Check, X } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardFooter, CardHeader } from '@/components/ui/card';
import { cn } from '@/utils';

export interface PricingFeature {
  id: string;
  label: React.ReactNode;
  included: boolean;
}

export interface PricingPlan {
  id: string;
  name: React.ReactNode;
  description?: React.ReactNode;
  /** Formatted price, for example "$19" or "Custom". */
  price: React.ReactNode;
  /** Billing period next to the price, for example "per month". */
  period?: React.ReactNode;
  features: PricingFeature[];
  /** Call to action, for example <Button asChild><Link href="/sign-up">…</Link></Button>. */
  action?: React.ReactNode;
  highlighted?: boolean;
  /** Badge text on the highlighted plan, for example "Most popular". */
  badge?: React.ReactNode;
}

export interface PricingTableLabels {
  /** Announced before an included feature, for example "Included". */
  included: string;
  /** Announced before a missing feature, for example "Not included". */
  notIncluded: string;
}

export interface PricingTableProps {
  plans: PricingPlan[];
  labels: PricingTableLabels;
  title?: React.ReactNode;
  description?: React.ReactNode;
  /** Content between the header and the plans, such as a billing-period toggle. */
  controls?: React.ReactNode;
  className?: string;
}

const gridColumns: Record<number, string> = {
  1: 'mx-auto max-w-md',
  2: 'md:grid-cols-2 mx-auto max-w-4xl',
  3: 'md:grid-cols-2 lg:grid-cols-3',
};

export function PricingTable({
  plans,
  labels,
  title,
  description,
  controls,
  className,
}: PricingTableProps) {
  const idPrefix = React.useId();
  const headingId = `${idPrefix}-heading`;

  return (
    <section
      aria-labelledby={title ? headingId : undefined}
      className={cn('flex flex-col gap-8', className)}
    >
      {title || description ? (
        <header className="mx-auto flex max-w-2xl flex-col gap-2 text-center">
          {title ? (
            <h2 id={headingId} className="text-3xl font-semibold tracking-tight text-foreground">
              {title}
            </h2>
          ) : null}
          {description ? <p className="text-base text-muted-foreground">{description}</p> : null}
        </header>
      ) : null}

      {controls ? <div className="flex justify-center">{controls}</div> : null}

      <ul
        role="list"
        className={cn(
          'grid w-full grid-cols-1 gap-6',
          gridColumns[plans.length] ?? 'md:grid-cols-2 xl:grid-cols-4'
        )}
      >
        {plans.map(plan => {
          const planHeadingId = `${idPrefix}-${plan.id}`;
          return (
            <li key={plan.id} className="min-w-0">
              <Card
                role="group"
                aria-labelledby={planHeadingId}
                data-highlighted={plan.highlighted ? '' : undefined}
                className={cn(
                  'relative h-full',
                  plan.highlighted && 'border-primary shadow-md ring-1 ring-primary'
                )}
              >
                <CardHeader className="gap-2">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <h3 id={planHeadingId} className="text-lg font-semibold text-card-foreground">
                      {plan.name}
                    </h3>
                    {plan.highlighted && plan.badge ? <Badge>{plan.badge}</Badge> : null}
                  </div>
                  {plan.description ? <CardDescription>{plan.description}</CardDescription> : null}
                  <p className="mt-2 flex flex-wrap items-baseline gap-x-1">
                    <span className="text-4xl font-semibold tracking-tight text-card-foreground tabular-nums">
                      {plan.price}
                    </span>
                    {plan.period ? (
                      <span className="text-sm text-muted-foreground">{plan.period}</span>
                    ) : null}
                  </p>
                </CardHeader>
                <CardContent className="flex-1">
                  <ul role="list" className="flex flex-col gap-3 text-sm">
                    {plan.features.map(feature => (
                      <li
                        key={feature.id}
                        className={cn(
                          'flex items-start gap-2',
                          feature.included ? 'text-card-foreground' : 'text-muted-foreground'
                        )}
                      >
                        {feature.included ? (
                          <Check
                            className="mt-0.5 size-4 shrink-0 text-success"
                            aria-hidden="true"
                          />
                        ) : (
                          <X className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
                        )}
                        <span>
                          <span className="sr-only">
                            {feature.included ? labels.included : labels.notIncluded}:{' '}
                          </span>
                          {feature.label}
                        </span>
                      </li>
                    ))}
                  </ul>
                </CardContent>
                {plan.action ? (
                  <CardFooter className="flex-col items-stretch [&>*]:w-full">
                    {plan.action}
                  </CardFooter>
                ) : null}
              </Card>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
