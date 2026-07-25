import { act } from '@testing-library/react';
import { usePreferencesStore } from '@/features/preferences/store/preferences-store';

describe('preferences store', () => {
  beforeEach(() => {
    usePreferencesStore.setState({ theme: 'system' });
  });

  it('updates the browser-only theme preference', () => {
    act(() => usePreferencesStore.getState().setTheme('dark'));

    expect(usePreferencesStore.getState().theme).toBe('dark');
  });
});
