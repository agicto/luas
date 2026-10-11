# Settings Page Block

A sectioned settings layout: page heading with optional actions, a section navigation (a sticky
sidebar on wide screens, a horizontal strip on phones), and one labelled card per section with an
optional footer for actions. Sections with `tone: 'danger'` get the destructive border for
irreversible settings.

```bash
cd api && go run ./cmd/fastbuild plugin:add block-settings-page
```

Installs `web/src/components/blocks/settings-page/` (component and test). It is a Server
Component-safe block: no state, no data fetching, no hard-coded copy. Forms are passed in.

## Usage

```tsx
import { SettingsPage } from '@/components/blocks/settings-page';
import { getT } from '@/i18n/server';
import { ProfileForm } from '@/features/profile/components/profile-form';
import { DeleteAccountButton } from '@/features/profile/components/delete-account-button';

export default async function AccountSettingsPage() {
  const t = await getT('settings');
  return (
    <SettingsPage
      title={t('title')}
      description={t('description')}
      navLabel={t('sectionsLabel')}
      sections={[
        { id: 'profile', title: t('profile.title'), description: t('profile.description'), content: <ProfileForm /> },
        { id: 'danger', title: t('danger.title'), tone: 'danger', content: <p>{t('danger.description')}</p>, footer: <DeleteAccountButton /> },
      ]}
    />
  );
}
```

| Prop | Type | Notes |
|---|---|---|
| `title`, `description`, `actions` | `ReactNode` | Page header |
| `navLabel` | `string` | Accessible name of the section navigation |
| `sections` | `SettingsSection[]` | `id` (anchor), `title`, `description`, `navLabel`, `content`, `footer`, `tone` |
| `activeSectionId` | `string` | Marks the navigation item with `aria-current="location"` |
| `headingLevel` | `'h1' \| 'h2'` | Use `h2` when the page already has an `h1`; section headings follow |

See `INTEGRATION.md` for placement, i18n, and verification.
