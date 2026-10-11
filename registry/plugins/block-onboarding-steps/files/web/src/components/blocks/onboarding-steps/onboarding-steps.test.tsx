import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { OnboardingSteps, type OnboardingStep } from './index';

const steps: OnboardingStep[] = [
  { id: 'profile', title: 'Complete your profile', status: 'complete' },
  {
    id: 'invite',
    title: 'Invite your team',
    description: 'Teammates get an email invitation.',
    status: 'current',
    action: <a href="/console/members">Invite members</a>,
  },
  {
    id: 'project',
    title: 'Create a project',
    status: 'upcoming',
    action: <a href="/console/projects">Create project</a>,
  },
];

const labels = {
  progress: '1 of 3 steps complete',
  complete: 'Completed',
  current: 'Current step',
  upcoming: 'Not started',
};

describe('OnboardingSteps block', () => {
  it('renders a labelled region with an ordered list of steps', () => {
    render(<OnboardingSteps title="Get started" steps={steps} labels={labels} />);

    const region = screen.getByRole('region', { name: 'Get started' });
    const items = within(within(region).getByRole('list')).getAllByRole('listitem');
    expect(items).toHaveLength(3);
    expect(screen.getByText('Teammates get an email invitation.')).toBeInTheDocument();
  });

  it('exposes progress and the current step to assistive technology', () => {
    render(<OnboardingSteps title="Get started" steps={steps} labels={labels} />);

    const progress = screen.getByRole('progressbar', { name: '1 of 3 steps complete' });
    expect(progress).toHaveAttribute('aria-valuenow', '1');
    expect(progress).toHaveAttribute('aria-valuemax', '3');

    const current = screen.getByRole('heading', { name: 'Invite your team (Current step)' });
    expect(current.closest('li')).toHaveAttribute('aria-current', 'step');
    expect(
      screen.getByRole('heading', { name: 'Complete your profile (Completed)' })
    ).toBeInTheDocument();
  });

  it('shows the action only on the current step', () => {
    render(<OnboardingSteps title="Get started" steps={steps} labels={labels} />);

    expect(screen.getByRole('link', { name: 'Invite members' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Create project' })).not.toBeInTheDocument();
  });
});
