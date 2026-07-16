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

import { buttonVariants } from '@/components/ui/button';
import { ROUTES } from '@/constants/routes';
import { cn } from '@/utils';

/**
 * Site Header Navigation Component
 *
 * Static public navigation. Authenticated state belongs to the console shell,
 * so the public site does not hydrate the auth store on first load.
 */
export async function SiteHeaderNav() {
  const t = await getTranslations();

  return (
    <div className="flex items-center gap-2 sm:gap-4">
      <Link
        href={ROUTES.AUTH.LOGIN}
        className={cn(buttonVariants({ variant: 'ghost', size: 'sm' }), 'interactive rounded-md')}
      >
        {t('auth.signIn')}
      </Link>
      <Link
        href={ROUTES.AUTH.REGISTER}
        className={cn(buttonVariants({ size: 'sm' }), 'interactive rounded-md max-sm:hidden')}
      >
        {t('auth.getStarted')}
      </Link>
    </div>
  );
}
