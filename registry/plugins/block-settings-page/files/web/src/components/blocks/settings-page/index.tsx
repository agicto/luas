/**
 * @block SettingsPage
 * @description A sectioned settings layout: a page header, a section navigation that becomes a
 * horizontal strip on small screens, and one framed card per section with an optional footer for
 * actions. All copy arrives through props; forms are passed in as section content. The block keeps
 * no state and fetches nothing, so it is safe in Server Components.
 */
import * as React from 'react';

import { Card, CardContent, CardDescription, CardFooter, CardHeader } from '@/components/ui/card';
import { cn } from '@/utils';

export interface SettingsSection {
  /** Used as the anchor id; must be unique on the page. */
  id: string;
  title: React.ReactNode;
  description?: React.ReactNode;
  /** Short label in the navigation; defaults to the title. */
  navLabel?: React.ReactNode;
  content: React.ReactNode;
  /** Actions such as a save button, rendered under a divider. */
  footer?: React.ReactNode;
  /** Use for irreversible settings such as deletion; frames the card with the destructive border. */
  tone?: 'default' | 'danger';
}

export interface SettingsPageProps {
  title: React.ReactNode;
  description?: React.ReactNode;
  /** Accessible name of the section navigation. */
  navLabel: string;
  sections: SettingsSection[];
  /** Highlights a section in the navigation, for example from the current URL hash or route. */
  activeSectionId?: string;
  /** Content aligned with the title, such as a primary action. */
  actions?: React.ReactNode;
  headingLevel?: 'h1' | 'h2';
  className?: string;
}

export function SettingsPage({
  title,
  description,
  navLabel,
  sections,
  activeSectionId,
  actions,
  headingLevel = 'h1',
  className,
}: SettingsPageProps) {
  const Heading = headingLevel;
  const SectionHeading = headingLevel === 'h1' ? 'h2' : 'h3';
  const idPrefix = React.useId();

  return (
    <div className={cn('flex flex-col gap-6', className)}>
      <header className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 flex-col gap-1">
          <Heading className="text-2xl font-semibold tracking-tight text-foreground">
            {title}
          </Heading>
          {description ? <p className="text-sm text-muted-foreground">{description}</p> : null}
        </div>
        {actions ? <div className="flex shrink-0 flex-wrap gap-2">{actions}</div> : null}
      </header>

      <div className="grid gap-6 md:grid-cols-[12rem_minmax(0,1fr)] lg:grid-cols-[14rem_minmax(0,1fr)]">
        <nav aria-label={navLabel} className="min-w-0 md:sticky md:top-6 md:self-start">
          <ul className="-mx-1 flex gap-1 overflow-x-auto px-1 pb-1 md:mx-0 md:flex-col md:overflow-visible md:px-0 md:pb-0">
            {sections.map(section => {
              const active = section.id === activeSectionId;
              return (
                <li key={section.id} className="shrink-0">
                  <a
                    href={`#${section.id}`}
                    aria-current={active ? 'location' : undefined}
                    className={cn(
                      'focus-ring block rounded-md px-3 py-2 text-sm font-medium whitespace-nowrap text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground',
                      active && 'bg-accent text-accent-foreground'
                    )}
                  >
                    {section.navLabel ?? section.title}
                  </a>
                </li>
              );
            })}
          </ul>
        </nav>

        <div className="flex min-w-0 flex-col gap-6">
          {sections.map(section => {
            const headingId = `${idPrefix}-${section.id}`;
            return (
              <section
                key={section.id}
                id={section.id}
                aria-labelledby={headingId}
                className="scroll-mt-6"
              >
                <Card className={cn('gap-5', section.tone === 'danger' && 'border-destructive')}>
                  <CardHeader className="gap-1">
                    <SectionHeading
                      id={headingId}
                      className={cn(
                        'text-base font-semibold leading-none text-card-foreground',
                        section.tone === 'danger' && 'text-error'
                      )}
                    >
                      {section.title}
                    </SectionHeading>
                    {section.description ? (
                      <CardDescription>{section.description}</CardDescription>
                    ) : null}
                  </CardHeader>
                  <CardContent className="flex flex-col gap-4">{section.content}</CardContent>
                  {section.footer ? (
                    <CardFooter className="flex-wrap justify-end gap-2 border-t">
                      {section.footer}
                    </CardFooter>
                  ) : null}
                </Card>
              </section>
            );
          })}
        </div>
      </div>
    </div>
  );
}
