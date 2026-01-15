import { PropsWithChildren } from 'react';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { SiteHeaderNav } from '@/components/features/site/site-header-nav';

/**
 * Site layout for public pages
 * Includes navigation bar and footer
 */
export default function SiteLayout({ children }: PropsWithChildren) {
  return (
    <div className="flex min-h-screen flex-col">
      {/* Navigation bar */}
      <header className="sticky top-0 z-50 border-b bg-background/80 backdrop-blur-sm">
        <div className="container flex h-16 items-center justify-between">
          <div className="flex items-center gap-6">
            {/* Logo */}
            <Link href="/" className="flex items-center space-x-2">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="h-6 w-6 text-primary"
              >
                <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10" />
              </svg>
              <span className="text-xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                Llamacto
              </span>
            </Link>

            {/* Navigation links */}
            <nav className="hidden md:flex items-center gap-6">
              <Link href="/" className="text-sm font-medium transition-colors hover:text-primary">
                Home
              </Link>
              <Link href="/features" className="text-sm font-medium transition-colors hover:text-primary">
                Features
              </Link>
              <Link href="/docs" className="text-sm font-medium transition-colors hover:text-primary">
                Documentation
              </Link>
              <Link href="/pricing" className="text-sm font-medium transition-colors hover:text-primary">
                Pricing
              </Link>
              <Link href="/console" className="text-sm font-medium transition-colors hover:text-primary">
                Demo
              </Link>
            </nav>
          </div>

          {/* User actions - uses client component for auth-aware rendering */}
          <SiteHeaderNav />
        </div>
      </header>

      {/* Main content area */}
      <main className="flex-1">
        {children}
      </main>

      {/* Footer */}
      <footer className="border-t py-12 text-sm bg-muted/30">
        <div className="container grid gap-8 md:grid-cols-2 lg:grid-cols-4">
          <div className="space-y-4">
            <h3 className="text-lg font-medium">Product</h3>
            <ul className="grid gap-2">
              <li><Link href="/features" className="opacity-70 hover:opacity-100">Features</Link></li>
              <li><Link href="/pricing" className="opacity-70 hover:opacity-100">Pricing</Link></li>
              <li><Link href="/console" className="opacity-70 hover:opacity-100">Demo</Link></li>
            </ul>
          </div>
          <div className="space-y-4">
            <h3 className="text-lg font-medium">Resources</h3>
            <ul className="grid gap-2">
              <li><Link href="/docs" className="opacity-70 hover:opacity-100">Documentation</Link></li>
              <li><Link href="/examples" className="opacity-70 hover:opacity-100">Examples</Link></li>
              <li><Link href="/blog" className="opacity-70 hover:opacity-100">Blog</Link></li>
            </ul>
          </div>
          <div className="space-y-4">
            <h3 className="text-lg font-medium">Support</h3>
            <ul className="grid gap-2">
              <li><Link href="/help" className="opacity-70 hover:opacity-100">Help Center</Link></li>
              <li><Link href="/community" className="opacity-70 hover:opacity-100">Community</Link></li>
              <li><Link href="/contact" className="opacity-70 hover:opacity-100">Contact</Link></li>
            </ul>
          </div>
          <div className="space-y-4">
            <h3 className="text-lg font-medium">Company</h3>
            <ul className="grid gap-2">
              <li><Link href="/about" className="opacity-70 hover:opacity-100">About Us</Link></li>
              <li><Link href="/careers" className="opacity-70 hover:opacity-100">Careers</Link></li>
              <li><Link href="/press" className="opacity-70 hover:opacity-100">Press Kit</Link></li>
            </ul>
          </div>
        </div>
        <div className="container mt-8 flex flex-col items-center justify-between gap-4 border-t pt-8 md:flex-row">
          <p className="text-center text-sm opacity-70">
            &copy; {new Date().getFullYear()} Llamacto. All rights reserved.
          </p>
          <div className="flex gap-4">
            <Link href="/privacy" className="text-sm opacity-70 hover:opacity-100">Privacy Policy</Link>
            <Link href="/terms" className="text-sm opacity-70 hover:opacity-100">Terms of Service</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
