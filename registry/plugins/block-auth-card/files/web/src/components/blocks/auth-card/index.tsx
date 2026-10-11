/**
 * @block AuthCard
 * @description A sign-in or sign-up card shell: brand mark, heading, optional alternative
 * sign-in actions with a divider, a slot for the caller's form, a footer link line, and legal
 * text. It contains no authentication logic and fetches nothing; the caller passes the form, which
 * owns validation and submission. All copy arrives through props.
 */
import * as React from 'react';

import { Card, CardContent, CardDescription, CardFooter, CardHeader } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { cn } from '@/utils';

export interface AuthCardProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  /** Brand mark above the heading; decorative unless it carries its own accessible name. */
  logo?: React.ReactNode;
  /** The form, for example the existing sign-in form component. */
  children: React.ReactNode;
  /** Alternative sign-in buttons, such as single sign-on providers. */
  alternativeActions?: React.ReactNode;
  /** Divider text between the alternative actions and the form, for example "or". */
  dividerLabel?: string;
  /** Line under the form, for example a link to sign up or sign in instead. */
  footer?: React.ReactNode;
  /** Small print under the card, for example links to the terms and privacy policy. */
  legal?: React.ReactNode;
  headingLevel?: 'h1' | 'h2';
  className?: string;
}

export function AuthCard({
  title,
  description,
  logo,
  children,
  alternativeActions,
  dividerLabel,
  footer,
  legal,
  headingLevel = 'h1',
  className,
}: AuthCardProps) {
  const Heading = headingLevel;
  const headingId = React.useId();

  return (
    <section
      aria-labelledby={headingId}
      className={cn('mx-auto flex w-full max-w-sm flex-col gap-4', className)}
    >
      <Card className="gap-6">
        <CardHeader className="flex flex-col items-center gap-2 text-center">
          {logo ? <div className="mb-2 flex justify-center">{logo}</div> : null}
          <Heading
            id={headingId}
            className="text-xl font-semibold tracking-tight text-card-foreground"
          >
            {title}
          </Heading>
          {description ? <CardDescription>{description}</CardDescription> : null}
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          {alternativeActions ? (
            <>
              <div className="flex flex-col gap-2 [&>*]:w-full">{alternativeActions}</div>
              <div className="flex items-center gap-3">
                <Separator className="flex-1" />
                {dividerLabel ? (
                  <span className="text-xs text-muted-foreground uppercase">{dividerLabel}</span>
                ) : null}
                <Separator className="flex-1" />
              </div>
            </>
          ) : null}
          {children}
        </CardContent>
        {footer ? (
          <CardFooter className="justify-center text-center text-sm text-muted-foreground">
            {footer}
          </CardFooter>
        ) : null}
      </Card>
      {legal ? (
        <p className="px-2 text-center text-xs text-muted-foreground [&_a]:underline [&_a]:underline-offset-4">
          {legal}
        </p>
      ) : null}
    </section>
  );
}
