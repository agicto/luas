import { 
  Input, 
  SearchInput, 
  PasswordInput 
} from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Switch } from "@/components/ui/switch"
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "@/components/ui/select"
import { MailIcon, LockIcon } from "lucide-react"

export function FormShowcase() {
  return (
    <section className="space-y-8">
      <h2 className="text-2xl font-semibold">Form Controls</h2>
      
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-12">
        {/* Style Comparison */}
        <div className="space-y-6">
          <h3 className="text-lg font-medium">Specialized Inputs</h3>
          
          <div className="grid gap-6 p-6 border rounded-xl bg-card">
            <div className="grid gap-2">
              <Label>Search Input (Rounded-full)</Label>
              <SearchInput placeholder="Quick search components..." />
            </div>

            <div className="grid gap-2">
              <Label>Password Input</Label>
              <PasswordInput placeholder="Enter your password" />
            </div>

            <div className="grid gap-2">
              <Label>Input with Icons</Label>
              <Input 
                leftIcon={<MailIcon className="size-4" />} 
                placeholder="email@example.com" 
              />
            </div>
            
            <div className="grid gap-2">
              <Label className="text-destructive">Integrated Error Message</Label>
              <Input 
                errorText="This field is required and must be a valid email." 
                placeholder="email@example.com" 
              />
            </div>
            
            <div className="grid gap-2">
              <Label className="text-destructive">Explicit Error State (No Text)</Label>
              <Input error placeholder="Missing required field" />
            </div>
          </div>
        </div>

        {/* Binary Choices & States */}
        <div className="space-y-6">
          <h3 className="text-lg font-medium">Binary Choice & States</h3>
          
          <div className="grid gap-6 p-6 border rounded-xl bg-card">
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
              <Label className="text-destructive">Error State</Label>
              <Input aria-invalid="true" placeholder="Error field..." />
              <p className="text-xs text-destructive">Invalidated border and focus-ring colors.</p>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
