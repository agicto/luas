import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { DashboardStats, type StatItem } from './index';

const items: StatItem[] = [
  {
    id: 'revenue',
    label: 'Revenue',
    value: '$12,400',
    trend: { value: '+12%', direction: 'up', label: 'Up 12% from last month' },
    description: 'Last 30 days',
  },
  {
    id: 'errors',
    label: 'Error rate',
    value: '0.4%',
    trend: { value: '+0.1%', direction: 'up', label: 'Up 0.1 points', positive: false },
  },
  {
    id: 'seats',
    label: 'Seats',
    value: '48',
    trend: { value: '0%', direction: 'flat', label: 'No change' },
  },
];

describe('DashboardStats block', () => {
  it('renders a labelled list of metrics from props', () => {
    render(<DashboardStats title="Overview" description="This month" items={items} />);

    const region = screen.getByRole('region', { name: 'Overview' });
    const list = within(region).getByRole('list');
    expect(within(list).getAllByRole('listitem')).toHaveLength(3);
    expect(screen.getByText('$12,400')).toBeInTheDocument();
    expect(screen.getByText('Last 30 days')).toBeInTheDocument();
    expect(screen.getByText('This month')).toBeInTheDocument();
  });

  it('announces trends in words and colors them by meaning, not direction', () => {
    render(<DashboardStats items={items} />);

    expect(screen.getByText('Up 12% from last month')).toHaveClass('sr-only');
    expect(screen.getByText('Up 12% from last month').parentElement).toHaveClass('text-success');
    expect(screen.getByText('Up 0.1 points').parentElement).toHaveClass('text-error');
    expect(screen.getByText('No change').parentElement).toHaveClass('text-muted-foreground');
  });

  it('marks the region busy and announces the loading label while loading', () => {
    render(
      <DashboardStats title="Overview" items={items} loading loadingLabel="Loading metrics" />
    );

    const region = screen.getByRole('region', { name: 'Overview' });
    expect(region).toHaveAttribute('aria-busy', 'true');
    expect(screen.getByRole('status')).toHaveTextContent('Loading metrics');
    expect(screen.queryByText('$12,400')).not.toBeInTheDocument();
  });
});
