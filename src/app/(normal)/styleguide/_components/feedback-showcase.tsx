"use client"

import * as React from "react"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import { 
  Tooltip, 
  TooltipContent, 
  TooltipProvider, 
  TooltipTrigger 
} from "@/components/ui/tooltip"
import { InfoIcon, AlertCircle } from "lucide-react"

export function FeedbackShowcase() {
  return (
    <section className="space-y-8">
      <h2 className="text-2xl font-semibold">Feedback & Interaction</h2>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-12">
        <div className="space-y-4">
          <h3 className="text-lg font-medium">Messaging</h3>
          <div className="space-y-4">
            <Alert>
              <InfoIcon className="h-4 w-4" />
              <AlertTitle>System Update</AlertTitle>
              <AlertDescription>
                A new version of the styleguide is available for review.
              </AlertDescription>
            </Alert>
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Critical Error</AlertTitle>
              <AlertDescription>
                Failed to apply spring easing to the main thread.
              </AlertDescription>
            </Alert>
          </div>
        </div>

        <div className="space-y-8">
          <div className="space-y-4">
            <h3 className="text-lg font-medium">Skeleton Loaders</h3>
            <div className="flex items-center space-x-4 p-6 border rounded-xl bg-card">
              <Skeleton className="size-12 rounded-full" />
              <div className="space-y-2">
                <Skeleton className="h-4 w-[250px]" />
                <Skeleton className="h-4 w-[200px]" />
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <h3 className="text-lg font-medium">Contextual Feedback</h3>
            <div className="p-6 border rounded-xl bg-card flex justify-center">
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline">Hover for Details</Button>
                  </TooltipTrigger>
                  <TooltipContent className="glass-panel text-xs">
                    <p>Spring-animated premium tooltip</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
