import type { SettingsMessages } from './zh-Hans';

const messages: SettingsMessages = {
  title: 'Settings',
  tabs: {
    general: 'General settings',
    notifications: 'Notifications',
    security: 'Security',
    api: 'API',
  },
  system: {
    title: 'System information',
    description: 'View and update the basic system settings.',
    companyName: 'Company name',
    companyPlaceholder: 'Enter a company name',
    companyDefault: 'Example Technology Ltd.',
    websiteUrl: 'Website URL',
    websitePlaceholder: 'Enter a website address',
    supportEmail: 'Support email',
    supportEmailPlaceholder: 'Enter a support email',
    save: 'Save system settings',
  },
  display: {
    title: 'Display settings',
    description: 'Customize system display preferences.',
    darkMode: 'Dark mode',
    darkModeDescription: 'Enable dark mode across the system.',
    autoTheme: 'Automatic theme',
    autoThemeDescription: 'Follow the operating system theme setting.',
    save: 'Save display preferences',
  },
  notifications: {
    title: 'Notification settings',
    description: 'Configure how you receive notifications.',
    email: 'Email notifications',
    emailDescription: 'Receive business and system notifications by email.',
    sms: 'SMS notifications',
    smsDescription: 'Receive text alerts for important events.',
    browserPush: 'Browser push',
    browserPushDescription: 'Allow the browser to send push notifications.',
    save: 'Save notification settings',
  },
  security: {
    title: 'Security settings',
    description: 'Manage account security and access permissions.',
    twoFactor: 'Two-factor authentication',
    twoFactorDescription: 'Use two-factor authentication to protect the account.',
    sessionTimeout: 'Session timeout (minutes)',
    changePassword: 'Change password',
    save: 'Save security settings',
  },
  api: {
    title: 'API settings',
    description: 'Manage API keys and access permissions.',
    apiKey: 'API key',
    regenerate: 'Regenerate',
    apiKeyWarning: 'Keep this API key secure. It has full account access.',
    enable: 'Enable API access',
    enableDescription: 'Allow system data to be accessed through the API.',
    save: 'Save API settings',
  },
};

export default messages;
