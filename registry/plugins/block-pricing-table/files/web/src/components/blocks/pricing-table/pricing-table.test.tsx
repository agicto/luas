import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { PricingTable, type PricingPlan } from './index';

const plans: PricingPlan[] = [
  {
    id: 'starter',
    name: 'Starter',
    price: '$0',
    period: 'per month',
    features: [
      { id: 'projects', label: '3 projects', included: true },
      { id: 'sso', label: 'Single sign-on', included: false },
    ],
    action: <a href="/sign-up">Start free</a>,
  },
  {
    id: 'team',
    name: 'Team',
    description: 'For growing teams',
    price: '$19',
    period: 'per seat per month',
    highlighted: true,
    badge: 'Most popular',
    features: [
      { id: 'projects', label: 'Unlimited projects', included: true },
      { id: 'sso', label: 'Single sign-on', included: true },
    ],
    action: <a href="/sign-up?plan=team">Choose Team</a>,
  },
];

const labels = { included: 'Included', notIncluded: 'Not included' };

describe('PricingTable block', () => {
  it('renders one labelled group per plan with price, period, and action', () => {
    render(<PricingTable title="Pricing" plans={plans} labels={labels} />);

    expect(screen.getByRole('region', { name: 'Pricing' })).toBeInTheDocument();
    const team = screen.getByRole('group', { name: 'Team' });
    expect(within(team).getByText('$19')).toBeInTheDocument();
    expect(within(team).getByText('per seat per month')).toBeInTheDocument();
    expect(within(team).getByRole('link', { name: 'Choose Team' })).toHaveAttribute(
      'href',
      '/sign-up?plan=team'
    );
  });

  it('highlights only the chosen plan and shows its badge', () => {
    render(<PricingTable plans={plans} labels={labels} />);

    const team = screen.getByRole('group', { name: 'Team' });
    const starter = screen.getByRole('group', { name: 'Starter' });
    expect(team).toHaveAttribute('data-highlighted');
    expect(within(team).getByText('Most popular')).toBeInTheDocument();
    expect(starter).not.toHaveAttribute('data-highlighted');
  });

  it('announces whether each feature is included instead of relying on icons', () => {
    render(<PricingTable plans={plans} labels={labels} />);

    const starter = screen.getByRole('group', { name: 'Starter' });
    const features = within(starter).getAllByRole('listitem');
    expect(features[0]).toHaveTextContent('Included: 3 projects');
    expect(features[1]).toHaveTextContent('Not included: Single sign-on');
  });
});
