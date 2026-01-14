"use client"

import * as React from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/components/ui/breadcrumb"
import { InfoIcon, AlertCircleIcon, CheckCircle2Icon } from "lucide-react"

export default function StyleguidePage() {
  return (
    <div className="container py-10 space-y-16">
      <section className="space-y-4">
        <div className="space-y-2">
          <h1 className="text-4xl font-bold tracking-tight">UI Styleguide</h1>
          <p className="text-xl text-text-subtle">
            A comprehensive showcase of all optimized UI components and design tokens.
          </p>
        </div>
        <Separator />
      </section>

      {/* Buttons & Actions */}
      <section className="space-y-6">
        <h2 className="text-2xl font-semibold">Buttons & Actions</h2>
        <div className="flex flex-wrap gap-4 items-center">
          <Button>Default Button</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="destructive">Destructive</Button>
          <Button variant="link">Link Button</Button>
          <Button disabled>Disabled</Button>
        </div>
        <div className="flex flex-wrap gap-4 items-center">
          <Button size="xs">Extra Small</Button>
          <Button size="sm">Small</Button>
          <Button size="default">Default</Button>
          <Button size="lg">Large</Button>
          <Button size="xl">Extra Large</Button>
          <Button size="2xl">2X Large</Button>
        </div>
        <div className="flex flex-wrap gap-4 items-center">
          <Button size="xs" isIcon variant="outline">
            <InfoIcon className="size-3" />
          </Button>
          <Button size="sm" isIcon variant="outline">
            <InfoIcon className="size-4" />
          </Button>
          <Button size="default" isIcon variant="outline">
            <InfoIcon className="size-4" />
          </Button>
          <Button size="lg" isIcon variant="outline">
            <InfoIcon className="size-5" />
          </Button>
          <Button size="xl" isIcon variant="outline">
            <InfoIcon className="size-6" />
          </Button>
          <Button size="2xl" isIcon variant="outline">
            <InfoIcon className="size-7" />
          </Button>
        </div>
      </section>

      {/* Badge */}
      <section className="space-y-6">
        <h2 className="text-2xl font-semibold">Badges</h2>
        <div className="flex flex-wrap gap-4">
          <Badge>Default</Badge>
          <Badge variant="secondary">Secondary</Badge>
          <Badge variant="outline">Outline</Badge>
          <Badge variant="destructive">Destructive</Badge>
        </div>
      </section>

      {/* Forms */}
      <section className="space-y-6">
        <h2 className="text-2xl font-semibold">Forms</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8 max-w-4xl">
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="input-demo">Input Field</Label>
              <Input id="input-demo" placeholder="Type something..." />
              <p className="text-xs text-text-muted">Standard text input with focus-ring.</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="input-error">Invalid Input</Label>
              <Input id="input-error" placeholder="Error state" aria-invalid="true" />
              <p className="text-xs text-destructive">This input has aria-invalid="true".</p>
            </div>
          </div>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label>Select Component</Label>
              <Select defaultValue="option-1">
                <SelectTrigger>
                  <SelectValue placeholder="Select an option" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="option-1">Option One</SelectItem>
                  <SelectItem value="option-2">Option Two</SelectItem>
                  <SelectItem value="option-3">Option Three</SelectItem>
                </SelectContent>
              </Select>
            </div>
            
            <div className="flex items-center space-x-2 border p-3 rounded-md bg-bg-subtle/50">
              <Checkbox id="terms" />
              <Label htmlFor="terms" className="cursor-pointer">Accept terms and conditions</Label>
            </div>

            <div className="flex items-center space-x-2 border p-3 rounded-md bg-bg-subtle/50">
              <Switch id="airplane-mode" />
              <Label htmlFor="airplane-mode" className="cursor-pointer">Airplane Mode</Label>
            </div>
          </div>
        </div>
      </section>

      {/* Data Display */}
      <section className="space-y-6">
        <h2 className="text-2xl font-semibold">Data Display</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <Card>
            <CardHeader>
              <CardTitle>Card Component</CardTitle>
              <CardDescription>A well-structured container for content.</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-text-subtle">
                Cards provide a clean way to group related information in a visually distinct way.
              </p>
            </CardContent>
            <CardFooter>
              <Button variant="outline" className="w-full">Action</Button>
            </CardFooter>
          </Card>

          <div className="space-y-4 md:col-span-2">
            <h3 className="text-lg font-medium">Tabs & Navigation</h3>
            <Tabs defaultValue="overview" className="w-full">
              <TabsList>
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="analytics">Analytics</TabsTrigger>
                <TabsTrigger value="settings">Settings</TabsTrigger>
              </TabsList>
              <TabsContent value="overview" className="p-4 border rounded-md mt-2">
                This is the overview tab content.
              </TabsContent>
              <TabsContent value="analytics" className="p-4 border rounded-md mt-2">
                This is the analytics tab content.
              </TabsContent>
              <TabsContent value="settings" className="p-4 border rounded-md mt-2">
                This is the settings tab content.
              </TabsContent>
            </Tabs>

            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem>
                  <BreadcrumbLink href="#">Home</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator />
                <BreadcrumbItem>
                  <BreadcrumbLink href="#">Styleguide</BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator />
                <BreadcrumbItem>
                  <BreadcrumbPage>Components</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
          </div>
        </div>
      </section>

      {/* Feedback & Interaction */}
      <section className="space-y-6">
        <h2 className="text-2xl font-semibold">Feedback & Interaction</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="space-y-4">
            <Alert>
              <InfoIcon className="h-4 w-4" />
              <AlertTitle>Heads up!</AlertTitle>
              <AlertDescription>
                You can add components to your app using the cli.
              </AlertDescription>
            </Alert>
            <Alert variant="destructive">
              <AlertCircleIcon className="h-4 w-4" />
              <AlertTitle>Error</AlertTitle>
              <AlertDescription>
                Your session has expired. Please log in again.
              </AlertDescription>
            </Alert>
          </div>

          <div className="space-y-6">
            <div className="space-y-2">
              <h3 className="text-sm font-medium">Skeleton Loaders</h3>
              <div className="flex items-center space-x-4">
                <Skeleton className="h-12 w-12 rounded-full" />
                <div className="space-y-2">
                  <Skeleton className="h-4 w-[250px]" />
                  <Skeleton className="h-4 w-[200px]" />
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <h3 className="text-sm font-medium">Tooltips</h3>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline">Hover for info</Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>This is a premium tooltip</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}
