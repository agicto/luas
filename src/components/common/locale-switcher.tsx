'use client';

import * as React from "react";
import { Globe, Check } from "lucide-react";
import { useLocale } from "@/hooks/use-locale";
import { locales, localeNames, isLocaleSwitcherEnabled, type Locale } from "@/i18n";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { cn } from "@/utils";

export function LanguageSwitcher() {
  const { locale, setLocale, isPending } = useLocale();

  if (!isLocaleSwitcherEnabled) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button 
          variant="ghost" 
          isIcon
          className="h-9 w-9 rounded-full hover:bg-muted/50 transition-colors"
          disabled={isPending}
        >
          <Globe className={cn("h-4.5 w-4.5 text-text-muted transition-transform duration-300", isPending && "animate-spin")} />
          <span className="sr-only">Toggle language</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[160px] rounded-xl p-1 shadow-premium border-border/50 animate-in fade-in zoom-in-95 duration-200">
        <div className="px-2 py-1.5 text-xs font-semibold text-text-muted uppercase tracking-wider">
          Select Language
        </div>
        {locales.map((loc) => (
          <DropdownMenuItem
            key={loc}
            onClick={() => setLocale(loc)}
            className={cn(
              "rounded-lg cursor-pointer flex items-center justify-between px-2 py-2 transition-colors",
              locale === loc ? "bg-primary/10 text-primary font-medium" : "text-text-subtle hover:bg-muted/50"
            )}
          >
            <span>{localeNames[loc]}</span>
            {locale === loc && <Check className="h-4 w-4" />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
