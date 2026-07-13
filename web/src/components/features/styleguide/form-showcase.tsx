"use client"

import { 
  ColorPicker,
  Input, 
  SearchInput, 
  PasswordInput 
} from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Switch } from "@/components/ui/switch"
import { Toggle } from "@/components/ui/toggle"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { DatePicker } from "@/components/ui/date-picker"
import { Textarea } from "@/components/ui/textarea"
import { Bold, Italic, Underline } from "lucide-react"
import * as React from "react"

export function FormShowcase() {
  const [date, setDate] = React.useState<Date | undefined>(new Date())
  const [dateTime, setDateTime] = React.useState<Date | undefined>(new Date())

  return (
    <section className="space-y-8">
      <h2 className="text-2xl font-semibold">Form Controls</h2>
      
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-12">
        {/* Specialized & Mixed Types */}
        <div className="space-y-6">
          <h3 className="text-lg font-medium">Specialized & Mixed Types</h3>
          
          <div className="grid gap-6 p-6 border rounded-xl bg-card">
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="styleguide-date">Premium Date Picker (with Year)</Label>
                <DatePicker id="styleguide-date" date={date} setDate={setDate} />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="styleguide-datetime">DateTime Picker (HMS)</Label>
                <DatePicker id="styleguide-datetime" showTime date={dateTime} setDate={setDateTime} />
              </div>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="styleguide-color">Color Picker</Label>
              <ColorPicker id="styleguide-color" />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="styleguide-file">File Upload</Label>
              <Input id="styleguide-file" type="file" className="cursor-pointer py-1.5 h-auto text-xs" />
            </div>

            <div className="grid gap-2 pt-4 border-t">
              <Label htmlFor="styleguide-search">Search Input (Pill)</Label>
              <SearchInput id="styleguide-search" placeholder="Quick search components..." />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="styleguide-password">Password Input</Label>
              <PasswordInput
                id="styleguide-password"
                placeholder="Enter your password"
                showPasswordLabel="Show password"
                hidePasswordLabel="Hide password"
              />
            </div>
          </div>
        </div>

        {/* Binary Choices & States */}
        {/* Binary Choices & Style Variants */}
        <div className="space-y-6">
          <h3 className="text-lg font-medium">Style Variants & States</h3>
          
          <div className="grid gap-6 p-6 border rounded-xl bg-card">
            <div className="grid gap-4">
              <div className="grid gap-2">
                <Label htmlFor="styleguide-outline">Outline Variant (Default)</Label>
                <Input id="styleguide-outline" placeholder="Standard outline input" />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="styleguide-filled">Filled Variant</Label>
                <Input id="styleguide-filled" variant="filled" placeholder="Subtle filled background" />
              </div>
            </div>

            <div className="grid gap-4 pt-4 border-t">
              <div className="grid gap-2">
                <Label htmlFor="styleguide-textarea">Textarea Outline</Label>
                <Textarea id="styleguide-textarea" placeholder="Multi-line input support..." />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="styleguide-textarea-filled">Textarea Filled</Label>
                <Textarea id="styleguide-textarea-filled" variant="filled" placeholder="Filled background textarea..." />
              </div>
            </div>
            <div className="flex items-start gap-4">
              <Checkbox id="cb1" className="mt-1" />
              <div className="grid gap-1.5 leading-none">
                <Label htmlFor="cb1" className="cursor-pointer">Checkbox Primary</Label>
                <p className="text-sm text-text-muted">Spring-animated scale and check animation.</p>
              </div>
            </div>

            <div className="flex items-center justify-between p-3 border rounded-lg bg-bg-subtle/30">
              <div className="grid gap-1">
                <Label htmlFor="sw1" className="cursor-pointer">Switch Feedback</Label>
                <p className="text-xs text-text-muted">Elastic transition on toggle.</p>
              </div>
              <Switch id="sw1" />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="styleguide-error" className="text-error">Error State</Label>
              <Input id="styleguide-error" aria-invalid="true" aria-describedby="styleguide-error-description" placeholder="Error field..." />
              <p id="styleguide-error-description" className="text-xs text-error">Invalidated border and focus-ring colors.</p>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="styleguide-textarea-error" className="text-error">Textarea Error State</Label>
              <Textarea id="styleguide-textarea-error" error errorText="This field is required" placeholder="Error state with message..." />
            </div>

            <div className="space-y-4 pt-4 border-t">
              <p className="text-sm font-medium">Selection Toggles</p>
              <div className="flex flex-wrap gap-4">
                <Toggle aria-label="Toggle italic" className="interactive-subtle">
                  <Italic className="h-4 w-4" />
                </Toggle>
                
                <ToggleGroup type="multiple" variant="outline">
                  <ToggleGroupItem value="bold" aria-label="Toggle bold" className="interactive-subtle">
                    <Bold className="h-4 w-4" />
                  </ToggleGroupItem>
                  <ToggleGroupItem value="italic" aria-label="Toggle italic">
                    <Italic className="h-4 w-4" />
                  </ToggleGroupItem>
                  <ToggleGroupItem value="underline" aria-label="Toggle underline">
                    <Underline className="h-4 w-4" />
                  </ToggleGroupItem>
                </ToggleGroup>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
