"use client"

import * as React from "react"
import { 
  Zap, 
  Activity, 
  CheckCircle, 
  Clock,
  Plus,
  Users,
  Settings,
  ArrowRight,
  Sparkles
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "@/components/ui/table"
import { StatCard } from "@/components/features/console/dashboard-stats"
import { DashboardChart } from "@/components/features/console/dashboard-chart"
import { ActivityTimeline, type ActivityItem } from "@/components/features/console/activity-timeline"

// Mock data for demonstration
const statsData = [
  {
    title: "Active Workflows",
    value: "12",
    description: "3 pending approval",
    trend: { value: "+2", isPositive: true },
    icon: Zap,
    variant: "primary" as const,
  },
  {
    title: "API Calls Today",
    value: "2,350",
    description: "Avg. 98 per hour",
    trend: { value: "+180", isPositive: true },
    icon: Activity,
    variant: "default" as const,
  },
  {
    title: "Success Rate",
    value: "99.2%",
    description: "Last 24 hours",
    trend: { value: "+0.1%", isPositive: true },
    icon: CheckCircle,
    variant: "success" as const,
  },
  {
    title: "Avg. Response Time",
    value: "142ms",
    description: "Within SLA",
    trend: { value: "-12ms", isPositive: true },
    icon: Clock,
    variant: "default" as const,
  },
]

const chartData = [
  { name: "Mon", value: 120 },
  { name: "Tue", value: 180 },
  { name: "Wed", value: 150 },
  { name: "Thu", value: 220 },
  { name: "Fri", value: 280 },
  { name: "Sat", value: 190 },
  { name: "Sun", value: 140 },
]

const activityItems: ActivityItem[] = [
  {
    id: "1",
    type: "success",
    title: "Workflow deployed successfully",
    description: "GPT-4 Translation Pipeline is now live",
    timestamp: "2 minutes ago",
    user: { name: "Admin", initials: "AD" },
  },
  {
    id: "2",
    type: "action",
    title: "New API key generated",
    description: "Production environment key",
    timestamp: "15 minutes ago",
    user: { name: "John Doe", initials: "JD" },
  },
  {
    id: "3",
    type: "warning",
    title: "Rate limit threshold reached",
    description: "85% of daily quota consumed",
    timestamp: "1 hour ago",
  },
  {
    id: "4",
    type: "config",
    title: "Settings updated",
    description: "Email notifications enabled",
    timestamp: "3 hours ago",
    user: { name: "Jane Smith", initials: "JS" },
  },
  {
    id: "5",
    type: "info",
    title: "System maintenance scheduled",
    description: "Planned for Sunday 2:00 AM UTC",
    timestamp: "Yesterday",
  },
]

const tableData = [
  { id: "WF-001", name: "Translation Pipeline", status: "Active", calls: "1,234", success: "99.8%" },
  { id: "WF-002", name: "Content Moderation", status: "Active", calls: "856", success: "98.5%" },
  { id: "WF-003", name: "Data Extraction", status: "Paused", calls: "432", success: "97.2%" },
  { id: "WF-004", name: "Sentiment Analysis", status: "Active", calls: "2,108", success: "99.1%" },
]

/**
 * Premium Console Dashboard Page
 * Demonstrates exemplary patterns for building console pages with:
 * - Premium visual design with gradients and animations
 * - Reusable component composition
 * - Interactive data displays
 */
export default function ConsolePage() {
  return (
    <div className="flex-1 space-y-8 p-6 pt-6">
      {/* Page Header with gradient accent */}
      <div className="relative overflow-hidden rounded-xl bg-linear-to-r from-primary/10 via-purple-500/5 to-transparent p-6 border border-primary/10">
        <div className="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNjAiIHZpZXdCb3g9IjAgMCA2MCA2MCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48ZyBmaWxsPSJub25lIiBmaWxsLXJ1bGU9ImV2ZW5vZGQiPjxwYXRoIGQ9Ik0zNiAxOGMzLjMxNCAwIDYgMi42ODYgNiA2cy0yLjY4NiA2LTYgNi02LTIuNjg2LTYtNiAyLjY4Ni02IDYtNiIgc3Ryb2tlPSJjdXJyZW50Q29sb3IiIHN0cm9rZS1vcGFjaXR5PSIuMDUiLz48L2c+PC9zdmc+')] opacity-50" />
        <div className="relative flex items-center justify-between">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <h1 className="text-3xl font-bold tracking-tight">
                Welcome back
              </h1>
              <Sparkles className="h-6 w-6 text-primary animate-pulse" />
            </div>
            <p className="text-text-muted">
              Here&apos;s an overview of your AI-powered workflows and integrations
            </p>
          </div>
          <Button className="gap-2">
            <Plus className="h-4 w-4" />
            New Workflow
          </Button>
        </div>
      </div>

      {/* Stats Cards Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {statsData.map((stat, index) => (
          <StatCard
            key={index}
            title={stat.title}
            value={stat.value}
            description={stat.description}
            trend={stat.trend}
            icon={stat.icon}
            variant={stat.variant}
          />
        ))}
      </div>

      {/* Charts and Activity Section */}
      <div className="grid gap-6 lg:grid-cols-5">
        <div className="lg:col-span-3 space-y-6">
          {/* API Usage Chart */}
          <DashboardChart 
            title="API Calls This Week" 
            data={chartData} 
            type="bar"
            height={180}
          />
          
          {/* Line Chart */}
          <DashboardChart 
            title="Response Time Trend" 
            data={chartData.map(d => ({ ...d, value: Math.floor(100 + Math.random() * 100) }))} 
            type="line"
            height={160}
          />
        </div>
        
        <div className="lg:col-span-2">
          <ActivityTimeline items={activityItems} />
        </div>
      </div>

      {/* Workflows Table */}
      <Card className="shadow-premium">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-lg font-semibold">Active Workflows</CardTitle>
          <Button variant="outline" size="sm" className="gap-2">
            View All
            <ArrowRight className="h-3 w-3" />
          </Button>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-24">ID</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">API Calls</TableHead>
                <TableHead className="text-right">Success Rate</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tableData.map((row) => (
                <TableRow key={row.id} className="interactive-bg transition-colors cursor-pointer">
                  <TableCell className="font-mono text-xs text-text-muted">{row.id}</TableCell>
                  <TableCell className="font-medium">{row.name}</TableCell>
                  <TableCell>
                    <Badge variant={row.status === "Active" ? "default" : "secondary"}>
                      {row.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right tabular-nums">{row.calls}</TableCell>
                  <TableCell className="text-right tabular-nums font-medium text-success">{row.success}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Quick Actions */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card className="group cursor-pointer transition-all hover:shadow-premium hover:-translate-y-0.5 hover:border-primary/30">
          <CardContent className="flex items-center gap-4 p-6">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
              <Plus className="h-6 w-6" />
            </div>
            <div>
              <h3 className="font-semibold">Create Workflow</h3>
              <p className="text-sm text-text-muted">Build a new AI pipeline</p>
            </div>
          </CardContent>
        </Card>

        <Card className="group cursor-pointer transition-all hover:shadow-premium hover:-translate-y-0.5 hover:border-primary/30">
          <CardContent className="flex items-center gap-4 p-6">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-muted text-text-muted transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
              <Users className="h-6 w-6" />
            </div>
            <div>
              <h3 className="font-semibold">Manage Users</h3>
              <p className="text-sm text-text-muted">Team member permissions</p>
            </div>
          </CardContent>
        </Card>

        <Card className="group cursor-pointer transition-all hover:shadow-premium hover:-translate-y-0.5 hover:border-primary/30">
          <CardContent className="flex items-center gap-4 p-6">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-muted text-text-muted transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
              <Settings className="h-6 w-6" />
            </div>
            <div>
              <h3 className="font-semibold">Settings</h3>
              <p className="text-sm text-text-muted">Configure your console</p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
