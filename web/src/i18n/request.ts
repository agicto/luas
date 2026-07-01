import { getRequestConfig } from 'next-intl/server';
import { cookies, headers } from 'next/headers';
import { defaultLocale } from './config';
import { isSupportedLocale, resolveRequestLocale } from './locale-resolution';
import { loadAllModules } from './loader';

export default getRequestConfig(async () => {
  const cookieStore = await cookies();
  const cookieLocale = cookieStore.get('locale')?.value;
  const acceptLanguage = isSupportedLocale(cookieLocale)
    ? null
    : (await headers()).get('accept-language');

  const locale = resolveRequestLocale({
    cookieLocale,
    acceptLanguage,
    defaultLocale,
  });

  return {
    locale,
    messages: await loadAllModules(locale),
  };
});
