import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { AuthCard } from './index';

function SignInForm({ onSubmit }: { onSubmit: () => void }) {
  return (
    <form
      aria-label="Sign in form"
      onSubmit={(event) => {
        event.preventDefault();
        onSubmit();
      }}
    >
      <label htmlFor="email">Email</label>
      <input id="email" type="email" />
      <button type="submit">Sign in</button>
    </form>
  );
}

describe('AuthCard block', () => {
  it('renders a labelled page heading and the caller form unchanged', () => {
    const onSubmit = vi.fn();
    render(
      <AuthCard
        title="Welcome back"
        description="Sign in to continue"
        footer={<a href="/sign-up">Create an account</a>}
        legal={<a href="/terms">Terms</a>}
      >
        <SignInForm onSubmit={onSubmit} />
      </AuthCard>
    );

    const region = screen.getByRole('region', { name: 'Welcome back' });
    expect(within(region).getByRole('heading', { level: 1 })).toHaveTextContent('Welcome back');
    expect(screen.getByText('Sign in to continue')).toBeInTheDocument();
    expect(screen.getByLabelText('Email')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Create an account' })).toHaveAttribute('href', '/sign-up');
    expect(screen.getByRole('link', { name: 'Terms' })).toBeInTheDocument();

    fireEvent.submit(screen.getByRole('form', { name: 'Sign in form' }));
    expect(onSubmit).toHaveBeenCalledOnce();
  });

  it('shows alternative actions with a decorative divider label', () => {
    render(
      <AuthCard
        title="Sign in"
        alternativeActions={<button type="button">Continue with single sign-on</button>}
        dividerLabel="or"
      >
        <SignInForm onSubmit={vi.fn()} />
      </AuthCard>
    );

    expect(screen.getByRole('button', { name: 'Continue with single sign-on' })).toBeInTheDocument();
    expect(screen.getByText('or')).toBeInTheDocument();
    expect(screen.queryByRole('separator')).not.toBeInTheDocument();
  });

  it('omits the divider without alternative actions and supports h2 headings', () => {
    render(
      <AuthCard title="Create your account" dividerLabel="or" headingLevel="h2">
        <SignInForm onSubmit={vi.fn()} />
      </AuthCard>
    );

    expect(screen.queryByText('or')).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 2, name: 'Create your account' })).toBeInTheDocument();
  });
});
