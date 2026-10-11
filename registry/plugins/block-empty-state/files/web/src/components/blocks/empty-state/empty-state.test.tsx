import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { EmptyState } from './index';

describe('EmptyState block', () => {
  it('renders a labelled region with copy and actions from props', () => {
    render(
      <EmptyState
        title="No projects yet"
        description="Create a project to get started."
        action={<button type="button">Create project</button>}
        secondaryAction={<a href="/docs">Read the guide</a>}
      />
    );

    const region = screen.getByRole('region', { name: 'No projects yet' });
    expect(region).toHaveAttribute('data-variant', 'empty');
    expect(screen.getByRole('heading', { level: 2, name: 'No projects yet' })).toBeInTheDocument();
    expect(screen.getByText('Create a project to get started.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create project' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Read the guide' })).toHaveAttribute('href', '/docs');
  });

  it('announces errors as an alert with the request detail', () => {
    render(
      <EmptyState
        variant="error"
        title="Could not load projects"
        detail="request_id: 1234"
        action={<button type="button">Try again</button>}
      />
    );

    const alert = screen.getByRole('alert');
    expect(alert).toHaveAttribute('data-variant', 'error');
    expect(alert).toHaveTextContent('Could not load projects');
    expect(alert).toHaveTextContent('request_id: 1234');
  });

  it('keeps the icon decorative and supports nested heading levels', () => {
    const { container } = render(
      <EmptyState variant="no-access" title="You do not have access" headingLevel="h3" bare />
    );

    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 3 })).toHaveTextContent('You do not have access');
    expect(container.querySelector('svg')?.closest('[aria-hidden="true"]')).not.toBeNull();
    expect(screen.getByRole('region')).not.toHaveClass('border-dashed');
  });
});
