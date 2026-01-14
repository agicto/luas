"use client"

import * as React from "react"
import { Button } from "@/components/ui/button"

export function DepthShowcase() {
  return (
    <section className="space-y-6">
      <h2 className="text-2xl font-semibold">Depth & Glassmorphism</h2>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-8 p-12 bg-muted/10 rounded-2xl relative overflow-hidden border">
        {/* Animated Background Decor */}
        <div className="absolute top-[-10%] left-[-10%] size-64 bg-primary/20 rounded-full blur-[100px] animate-pulse" />
        <div className="absolute bottom-[-10%] right-[-10%] size-64 bg-purple-500/20 rounded-full blur-[100px] animate-pulse" />
        
        <div className="glass-panel p-8 rounded-2xl flex flex-col gap-5 z-10 border-white/10">
          <div className="space-y-2">
            <h3 className="text-lg font-semibold">Glass Panel (Light)</h3>
            <p className="text-sm text-text-subtle leading-relaxed">
              Utilizes `backdrop-filter: blur` and `color-mix` for a translucent, modern feel that adapts to underlying content.
            </p>
          </div>
          <div className="flex gap-3">
            <Button variant="outline" size="sm">Secondary</Button>
            <Button size="sm">Primary</Button>
          </div>
        </div>

        <div className="glass-panel-dark p-8 rounded-2xl flex flex-col gap-5 z-10 border-white/5">
          <div className="space-y-2">
            <h3 className="text-lg font-semibold text-white">Glass Panel (Dark)</h3>
            <p className="text-sm text-white/60 leading-relaxed">
              Optimized for high-contrast environments or dark mode surfaces, maintaining readability through adjusted transparency.
            </p>
          </div>
          <div className="flex gap-3">
            <Button variant="outline" size="sm" className="border-white/20 text-white hover:bg-white/10">
              Ghost
            </Button>
            <Button size="sm" className="bg-white text-black hover:bg-white/90">
              Action
            </Button>
          </div>
        </div>
      </div>
    </section>
  )
}
