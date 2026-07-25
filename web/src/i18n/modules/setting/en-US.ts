const messages = {
  user: {
    title: 'Personal preferences',
    description: 'Locale and time zone for the current account.',
  },
  organization: {
    title: 'Organization preferences',
    description: 'Locale defaults for this organization.',
  },
  fields: {
    locale: 'Locale',
    localeDescription: 'Language and regional formatting.',
    timezone: 'Time zone',
    timezoneDescription: 'IANA time zone used for dates and schedules.',
  },
  options: {
    enUS: 'English (US)',
    zhHans: 'Simplified Chinese',
  },
  actions: {
    retry: 'Retry',
    reset: 'Reset to default',
    resetLocale: 'Reset locale to default',
    resetTimezone: 'Reset time zone to default',
    save: 'Save preferences',
  },
  messages: {
    saved: 'Preferences saved',
    reset: 'Preference reset',
  },
  errors: {
    forbidden: 'You cannot change these preferences.',
    generic: 'The preference could not be updated.',
    invalidResponse: 'The setting service returned an invalid response.',
    invalidValue: 'Check the selected locale or IANA time zone.',
    notFound: 'This setting is no longer available.',
    preconditionRequired: 'Refresh the page before changing this setting.',
    unavailable: 'The setting service is temporarily unavailable.',
    versionConflict: 'This setting changed elsewhere. The latest value has been loaded.',
  },
} as const;

export default messages;
export type SettingMessages = typeof messages;
