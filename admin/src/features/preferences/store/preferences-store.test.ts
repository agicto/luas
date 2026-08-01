import { act } from '@testing-library/react';
import { usePreferencesStore } from '@/features/preferences/store/preferences-store';

describe('preferences store', () => {
  beforeEach(() => {
    usePreferencesStore.setState({ sidebarOpen: true, theme: 'system' });
  });

  it('updates the browser-only theme preference', () => {
    act(() => usePreferencesStore.getState().setTheme('dark'));

    expect(usePreferencesStore.getState().theme).toBe('dark');
  });

  it('updates the browser-only sidebar preference', () => {
    act(() => usePreferencesStore.getState().setSidebarOpen(false));

    expect(usePreferencesStore.getState().sidebarOpen).toBe(false);
  });
});
