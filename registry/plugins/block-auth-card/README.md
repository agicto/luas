# Auth Card Block

A framed card for sign-in, sign-up, and password pages: brand mark, page heading, optional
alternative sign-in actions with a divider, a slot for your form, a footer line for switching
between sign-in and sign-up, and legal text under the card. It contains no authentication logic.

```bash
cd api && go run ./cmd/fastbuild plugin:add block-auth-card
```

Installs `web/src/components/blocks/auth-card/` (component and test). It is a Server
Component-safe block: no state, no data fetching, no hard-coded copy.

## Usage

```tsx
import Link from 'next/link';

import { AuthCard } from '@/components/blocks/auth-card';
import { getT } from '@/i18n/server';
import { LoginForm } from '@/features/auth/components/login-form'; // the existing form

export default async function LoginPage() {
  const t = await getT('auth');
  return (
    <AuthCard
      title={t('login.title')}
      description={t('login.description')}
      footer={<Link href="/register">{t('login.toRegister')}</Link>}
      legal={t('login.legal')}
    >
      <LoginForm />
    </AuthCard>
  );
}
```

| Prop | Type | Notes |
|---|---|---|
| `title`, `description` | `ReactNode` | `title` is the page heading and names the region |
| `children` | `ReactNode` | The form; it keeps its own validation, submission, and errors |
| `logo` | `ReactNode` | Brand mark above the heading |
| `alternativeActions`, `dividerLabel` | `ReactNode`, `string` | For example single sign-on buttons and "or" |
| `footer`, `legal` | `ReactNode` | Switch link and small print |
| `headingLevel` | `'h1' \| 'h2'` | Default `h1`; use `h2` when the layout renders the page heading |

See `INTEGRATION.md` for placement, i18n, and verification.
