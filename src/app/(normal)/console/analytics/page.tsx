import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

/**
 * Analytics page in console dashboard
 */
export default function AnalyticsPage() {
  return (
    <div className="flex-1 space-y-4 p-6 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">Analytics</h2>
      </div>
      
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle>User Activity</CardTitle>
            <CardDescription>User activity over the past 30 days</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-[300px] w-full bg-slate-100 dark:bg-slate-800 rounded-md flex items-center justify-center">
              <p className="text-sm text-muted-foreground">User Activity Chart</p>
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader>
            <CardTitle>Conversion Rate</CardTitle>
            <CardDescription>Registration to purchase funnel</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-[300px] w-full bg-slate-100 dark:bg-slate-800 rounded-md flex items-center justify-center">
              <p className="text-sm text-muted-foreground">Conversion Funnel Chart</p>
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader>
            <CardTitle>Geographic Distribution</CardTitle>
            <CardDescription>User geographic location distribution</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-[300px] w-full bg-slate-100 dark:bg-slate-800 rounded-md flex items-center justify-center">
              <p className="text-sm text-muted-foreground">Geographic Distribution Chart</p>
            </div>
          </CardContent>
        </Card>
      </div>
      
      <div className="grid gap-4 grid-cols-1">
        <Card>
          <CardHeader>
            <CardTitle>Performance Metrics</CardTitle>
            <CardDescription>System key performance metrics trends</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-[300px] w-full bg-slate-100 dark:bg-slate-800 rounded-md flex items-center justify-center">
              <p className="text-sm text-muted-foreground">Performance Metrics Trend</p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
