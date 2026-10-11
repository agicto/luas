import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { LandingHero, type LandingFeature } from './index';

const features: LandingFeature[] = [
  { id: 'fast', title: 'Fast setup', description: 'Start in minutes.', icon: <svg data-testid="icon" /> },
  { id: 'secure', title: 'Secure by default', description: 'Sessions and roles included.' },
  { id: 'own', title: 'Own the code', description: 'Everything is source in your repository.' },
];

describe('LandingHero block', () => {
  it('renders one page heading with copy and both calls to action', () => {
    render(
      <LandingHero
        eyebrow="New release"
        title="Build your product faster"
        description="A scaffold that ships with the boring parts done."
        primaryAction={<a href="/sign-up">Get started</a>}
        secondaryAction={<a href="/docs">Read the docs</a>}
      />
    );

    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1);
    const hero = screen.getByRole('region', { name: 'Build your product faster' });
    expect(within(hero).getByText('New release')).toBeInTheDocument();
    expect(within(hero).getByRole('link', { name: 'Get started' })).toHaveAttribute('href', '/sign-up');
    expect(within(hero).getByRole('link', { name: 'Read the docs' })).toBeInTheDocument();
  });

  it('renders the feature grid as a labelled list with decorative icons', () => {
    render(
      <LandingHero
        title="Build your product faster"
        features={features}
        featuresHeading="Why teams choose it"
        featuresHeadingHidden
      />
    );

    const section = screen.getByRole('region', { name: 'Why teams choose it' });
    expect(screen.getByRole('heading', { level: 2, name: 'Why teams choose it' })).toHaveClass('sr-only');
    expect(within(section).getAllByRole('listitem')).toHaveLength(3);
    expect(within(section).getAllByRole('heading', { level: 3 }).map((heading) => heading.textContent)).toEqual([
      'Fast setup',
      'Secure by default',
      'Own the code',
    ]);
    expect(screen.getByTestId('icon').parentElement).toHaveAttribute('aria-hidden', 'true');
  });

  it('places media beside the copy only when provided', () => {
    const { rerender } = render(<LandingHero title="Hero" />);
    expect(screen.getByRole('region', { name: 'Hero' })).toHaveClass('text-center');

    rerender(<LandingHero title="Hero" media={<div role="img" aria-label="Product screenshot" />} />);
    expect(screen.getByRole('region', { name: 'Hero' })).toHaveClass('lg:grid-cols-2');
    expect(screen.getByRole('img', { name: 'Product screenshot' })).toBeInTheDocument();
  });
});
