/**
 * @component SiteHeaderNav
 * @category Feature
 * @status Stable
 * @description The main navigation component for the public site header, supporting localized links and active state detection.
 * @usage Place in the site header for global navigation.
 * @example
 * <SiteHeaderNav />
 */
import Link from 'next/link';
import { getTranslations } from 'next-intl/server';

import { Button } from '@/components/ui/button';
import { ROUTES } from '@/constants/routes';

/**
 * Site Header Navigation Component
 * 
 * Static public navigation. Authenticated state belongs to the console shell,
 * so the public site does not hydrate the auth store on first load.
 */
export async function SiteHeaderNav() {
  const t = await getTranslations();

  return (
    <div className="flex items-center gap-4">
      <Link href={ROUTES.AUTH.LOGIN}>
        <Button variant="ghost" size="sm">{t('auth.signIn')}</Button>
      </Link>
      <Link href={ROUTES.AUTH.REGISTER}>
        <Button size="sm">{t('auth.getStarted')}</Button>
      </Link>
    </div>
  );
}
