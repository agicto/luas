import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { SettingsPage, type SettingsSection } from './index';

const sections: SettingsSection[] = [
  {
    id: 'profile',
    title: 'Profile',
    description: 'How others see you',
    content: <p>Profile form</p>,
    footer: <button type="button">Save profile</button>,
  },
  { id: 'notifications', title: 'Notifications', navLabel: 'Alerts', content: <p>Alert form</p> },
  { id: 'danger', title: 'Delete account', content: <p>Delete form</p>, tone: 'danger' },
];

describe('SettingsPage block', () => {
  it('renders the page heading and one labelled region per section', () => {
    render(<SettingsPage title="Settings" navLabel="Settings sections" sections={sections} />);

    expect(screen.getByRole('heading', { level: 1, name: 'Settings' })).toBeInTheDocument();
    const profile = screen.getByRole('region', { name: 'Profile' });
    expect(profile).toHaveAttribute('id', 'profile');
    expect(within(profile).getByText('How others see you')).toBeInTheDocument();
    expect(within(profile).getByRole('button', { name: 'Save profile' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 2, name: 'Delete account' })).toHaveClass('text-error');
  });

  it('links every navigation item to its section and marks the active one', () => {
    render(
      <SettingsPage
        title="Settings"
        navLabel="Settings sections"
        sections={sections}
        activeSectionId="notifications"
      />
    );

    const navigation = screen.getByRole('navigation', { name: 'Settings sections' });
    const links = within(navigation).getAllByRole('link');
    expect(links.map((link) => link.getAttribute('href'))).toEqual([
      '#profile',
      '#notifications',
      '#danger',
    ]);
    expect(within(navigation).getByRole('link', { name: 'Alerts' })).toHaveAttribute(
      'aria-current',
      'location'
    );
    expect(within(navigation).getByRole('link', { name: 'Profile' })).not.toHaveAttribute(
      'aria-current'
    );
  });

  it('shifts heading levels when embedded under an existing page heading', () => {
    render(
      <SettingsPage title="Workspace" navLabel="Sections" sections={sections} headingLevel="h2" />
    );

    expect(screen.getByRole('heading', { level: 2, name: 'Workspace' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 3, name: 'Profile' })).toBeInTheDocument();
  });
});
