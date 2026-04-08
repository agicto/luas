import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import { authConfig } from '@/config/auth';

function matchesPrefix(pathname: string, prefix: string): boolean {
  return pathname === prefix || pathname.startsWith(`${prefix}/`);
}

function isProtectedPath(pathname: string): boolean {
  return authConfig.protectedRoutes.some((route) => matchesPrefix(pathname, route));
}

function isPublicOnlyPath(pathname: string): boolean {
  return authConfig.publicOnlyRoutes.some((route) => matchesPrefix(pathname, route));
}

export function middleware(request: NextRequest) {
  const { pathname, search } = request.nextUrl;
  const hasSession = Boolean(request.cookies.get(authConfig.cookies.session)?.value);

  if (!hasSession && isProtectedPath(pathname)) {
    const loginUrl = new URL(authConfig.routes.login, request.url);
    loginUrl.searchParams.set('returnUrl', `${pathname}${search}`);
    return NextResponse.redirect(loginUrl);
  }

  if (hasSession && isPublicOnlyPath(pathname)) {
    return NextResponse.redirect(new URL(authConfig.routes.afterLogin, request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    '/console/:path*',
    '/styleguide/:path*',
    '/i18n-test/:path*',
    '/login',
    '/register',
  ],
};
