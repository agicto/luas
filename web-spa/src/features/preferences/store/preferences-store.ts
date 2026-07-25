import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type ThemePreference = 'dark' | 'light' | 'system';

interface PreferencesState {
  theme: ThemePreference;
  setTheme: (theme: ThemePreference) => void;
}

export const usePreferencesStore = create<PreferencesState>()(
  persist(
    (set) => ({
      theme: 'system',
      setTheme: (theme) => set({ theme }),
    }),
    {
      name: 'luas-spa-preferences',
      storage: createJSONStorage(() => window.localStorage),
      partialize: (state) => ({
        theme: state.theme,
      }),
    },
  ),
);
