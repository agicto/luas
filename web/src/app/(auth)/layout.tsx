import type { PropsWithChildren } from 'react';
import { Zap, Shield, Layers, Headphones } from 'lucide-react';
import { getMessages } from 'next-intl/server';
import { Logo } from '@/components/ui/icons';
import { Toaster } from '@/components/ui/sonner';
import { LanguageSwitcher } from '@/components/common';
import { CLIENT_MESSAGE_NAMESPACES } from '@/i18n/client-message-namespaces';
import { selectMessageNamespaces } from '@/i18n/message-selection';
import { RouteMessagesProvider } from '@/i18n/route-messages-provider';
import { getT } from '@/i18n/server';
import { QueryProvider } from '@/providers/query-provider';

const features = [
  { icon: Zap, key: 'feature1' },
  { icon: Shield, key: 'feature2' },
  { icon: Layers, key: 'feature3' },
  { icon: Headphones, key: 'feature4' },
] as const;

/**
 * Auth layout with enhanced decorative left panel
 */
export default async function AuthLayout({ children }: PropsWithChildren) {
  const [t, messages] = await Promise.all([getT('auth'), getMessages()]);
  const clientMessages = selectMessageNamespaces(
    messages,
    CLIENT_MESSAGE_NAMESPACES.auth
  );

  return (
    <div className="flex min-h-svh bg-bg-canvas">
      {/* Decorative Left Panel */}
      <div className="hidden lg:flex lg:w-1/2 relative overflow-hidden bg-linear-to-br from-primary via-primary/90 to-primary-deeper">
        {/* Animated Background Elements */}
        <div className="absolute inset-0">
          {/* Large gradient orbs */}
          <div className="absolute top-[-10%] left-[-5%] w-[50%] h-[50%] rounded-full bg-white/10 blur-3xl" />
          <div className="absolute bottom-[-20%] right-[-10%] w-[60%] h-[60%] rounded-full bg-primary-deeper/30 blur-3xl" />
          <div className="absolute top-[40%] left-[30%] w-[40%] h-[40%] rounded-full bg-white/5 blur-2xl" />
          
          {/* Decorative circles */}
          <div className="absolute top-[15%] right-[20%] w-32 h-32 rounded-full border border-white/10" />
          <div className="absolute top-[18%] right-[22%] w-24 h-24 rounded-full border border-white/5" />
          <div className="absolute bottom-[25%] left-[10%] w-48 h-48 rounded-full border border-white/10" />
          <div className="absolute bottom-[28%] left-[12%] w-40 h-40 rounded-full border border-white/5" />
          
          {/* Floating dots */}
          <div className="absolute top-[30%] right-[15%] w-2 h-2 rounded-full bg-white/30" />
          <div className="absolute top-[60%] right-[25%] w-3 h-3 rounded-full bg-white/20" />
          <div className="absolute bottom-[40%] left-[25%] w-2 h-2 rounded-full bg-white/25" />
        </div>
        
        {/* Grid Pattern Overlay */}
        <div 
          className="absolute inset-0 opacity-[0.03]"
          style={{
            backgroundImage: `linear-gradient(rgba(255,255,255,0.1) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.1) 1px, transparent 1px)`,
            backgroundSize: '50px 50px',
          }}
        />
        
        {/* Content Container - Vertically centered */}
        <div className="relative z-10 flex flex-col justify-center h-full px-10 xl:px-16 text-white">
          {/* Logo with glow effect */}
          <div className="flex items-center gap-3 mb-10">
            <div className="relative">
              <div className="absolute inset-0 bg-white/30 rounded-xl blur-xl scale-150" />
              <div className="relative flex h-14 w-14 items-center justify-center rounded-xl bg-white/20 backdrop-blur-sm border border-white/20 shadow-lg">
                <Logo className="h-8 w-8 text-white" />
              </div>
            </div>
            <span className="text-2xl font-bold tracking-tight drop-shadow-lg">Luas</span>
          </div>
          
          {/* Hero Text */}
          <div className="space-y-4 mb-10">
            <h1 className="text-3xl xl:text-4xl font-bold leading-tight drop-shadow-md">
              {t('heroTitle')}
            </h1>
            <p className="text-base xl:text-lg text-white/75 max-w-sm leading-relaxed">
              {t('heroSubtitle')}
            </p>
          </div>
          
          {/* Features Grid - Staggered layout */}
          <div className="grid grid-cols-2 gap-3 max-w-sm">
            {features.map((feature, index) => {
              const IconComponent = feature.icon;
              return (
                <div 
                  key={feature.key}
                  className="group flex items-center gap-2.5 p-3 rounded-xl bg-white/10 backdrop-blur-sm border border-white/10 transition-all duration-300 hover:bg-white/15 hover:border-white/20 hover:scale-[1.02] cursor-default"
                  style={{ animationDelay: `${index * 100}ms` }}
                >
                  <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-white/20 group-hover:bg-white/25 transition-colors">
                    <IconComponent className="h-4 w-4" />
                  </div>
                  <span className="text-sm font-medium text-white/90 leading-tight">
                    {t(feature.key)}
                  </span>
                </div>
              );
            })}
          </div>
          
        </div>
      </div>
      
      {/* Right Panel - Form Area */}
      <RouteMessagesProvider additionalMessages={clientMessages}>
        <div className="flex-1 flex flex-col relative bg-background">
          {/* Subtle gradient overlay for depth */}
          <div className="absolute inset-0 bg-gradient-to-br from-muted/30 via-transparent to-muted/20 pointer-events-none" />

          {/* Language Switcher */}
          <div className="absolute top-4 right-4 z-20">
            <LanguageSwitcher />
          </div>

          {/* Form Container */}
          <div className="relative flex-1 flex flex-col items-center justify-center p-6 md:p-8">
            {/* Mobile Logo */}
            <div className="lg:hidden mb-6 flex flex-col items-center space-y-3">
              <div className="flex items-center space-x-3">
                <Logo className="h-9 w-9 text-primary" />
                <span className="text-2xl font-bold bg-linear-to-r from-primary to-primary-deeper bg-clip-text text-transparent">
                  Luas
                </span>
              </div>
              <p className="text-center text-sm text-muted-foreground max-w-xs">
                {t('heroSubtitle')}
              </p>
            </div>

            {/* Auth content */}
            <div className="w-full max-w-xs md:max-w-sm animate-in fade-in slide-in-from-bottom-4 duration-500">
              <QueryProvider>{children}</QueryProvider>
            </div>
          </div>
          
          {/* Footer */}
          <div className="relative shrink-0 py-5 text-center text-xs text-muted-foreground">
            &copy; {new Date().getFullYear()} Luas. {t('allRightsReserved')}.
          </div>
        </div>
      </RouteMessagesProvider>
      <Toaster richColors position="top-right" />
    </div>
  );
}
