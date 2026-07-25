import { Languages, MonitorCog } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { changeLocale } from '@/i18n';
import type { SupportedLocale } from '@/i18n/resources';
import {
  type ThemePreference,
  usePreferencesStore,
} from '@/features/preferences/store/preferences-store';
import { cn } from '@/lib/cn';

const themes: ThemePreference[] = ['light', 'dark', 'system'];

export function PreferencesPage() {
  const { t, i18n } = useTranslation();
  const theme = usePreferencesStore((state) => state.theme);
  const setTheme = usePreferencesStore((state) => state.setTheme);

  return (
    <div className="page-stack">
      <header className="page-header">
        <p className="page-eyebrow">{t('preferences.eyebrow')}</p>
        <h1>{t('preferences.title')}</h1>
        <p>{t('preferences.description')}</p>
      </header>

      <div className="content-grid">
        <section className="panel" aria-labelledby="appearance-title">
          <div className="panel-header">
            <div className="flex items-start gap-3">
              <span className="icon-surface" aria-hidden="true">
                <MonitorCog className="size-4" />
              </span>
              <div>
                <h2 id="appearance-title" className="panel-title">
                  {t('preferences.appearanceTitle')}
                </h2>
                <p className="panel-description">{t('preferences.appearanceDescription')}</p>
              </div>
            </div>
          </div>
          <div className="panel-body">
            <span className="field-label">{t('preferences.theme')}</span>
            <div className="segmented-control" role="group" aria-label={t('preferences.theme')}>
              {themes.map((value) => (
                <Button
                  key={value}
                  className={cn(
                    'flex-1',
                    theme === value &&
                      'bg-background text-brand-strong shadow-sm hover:bg-background hover:text-brand-strong',
                  )}
                  variant="ghost"
                  size="sm"
                  aria-pressed={theme === value}
                  onClick={() => setTheme(value)}
                >
                  {t(`preferences.${value}`)}
                </Button>
              ))}
            </div>
          </div>
        </section>

        <section className="panel" aria-labelledby="language-title">
          <div className="panel-header">
            <div className="flex items-start gap-3">
              <span className="icon-surface" aria-hidden="true">
                <Languages className="size-4" />
              </span>
              <div>
                <h2 id="language-title" className="panel-title">
                  {t('preferences.languageTitle')}
                </h2>
                <p className="panel-description">{t('preferences.languageDescription')}</p>
              </div>
            </div>
          </div>
          <div className="panel-body">
            <label className="field-label" htmlFor="locale">
              {t('preferences.language')}
            </label>
            <select
              id="locale"
              className="select-input"
              value={i18n.language}
              onChange={(event) => void changeLocale(event.target.value as SupportedLocale)}
            >
              <option value="en">{t('preferences.english')}</option>
              <option value="zh-CN">{t('preferences.chinese')}</option>
            </select>
            <p className="mt-3 text-xs text-subtle">{t('preferences.storageNote')}</p>
          </div>
        </section>
      </div>
    </div>
  );
}
