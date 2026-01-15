// i18n Modules barrel export
import * as common from './common';
import * as auth from './auth';
import * as nav from './nav';
import * as settings from './settings';
import * as errors from './errors';
import * as metadata from './metadata';
import * as dashboard from './dashboard';

import type { Locale } from '../config';

// Module definitions
const modules = {
  common,
  auth,
  nav,
  settings,
  errors,
  metadata,
  dashboard,
} as const;

// Locale key to export name mapping
const localeToExport: Record<Locale, 'zhHans' | 'enUS'> = {
  'zh-Hans': 'zhHans',
  'en-US': 'enUS',
};

/**
 * Get all messages for a specific locale
 */
export function getMessages(locale: Locale) {
  const exportKey = localeToExport[locale];
  
  return {
    common: modules.common[exportKey],
    auth: modules.auth[exportKey],
    nav: modules.nav[exportKey],
    settings: modules.settings[exportKey],
    errors: modules.errors[exportKey],
    metadata: modules.metadata[exportKey],
    dashboard: modules.dashboard[exportKey],
  };
}

// Type definitions
export type Messages = ReturnType<typeof getMessages>;
