import { resolveDarkTheme } from '@/features/preferences/hooks/use-resolved-theme';

describe('resolveDarkTheme', () => {
  it('resolves explicit and system theme preferences', () => {
    expect(resolveDarkTheme('light', true)).toBe(false);
    expect(resolveDarkTheme('dark', false)).toBe(true);
    expect(resolveDarkTheme('system', true)).toBe(true);
    expect(resolveDarkTheme('system', false)).toBe(false);
  });
});
