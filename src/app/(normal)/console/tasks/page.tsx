'use client';

import { useState } from 'react';
import { 
  Plus, 
  Search, 
  MoreHorizontal, 
  CheckCircle2, 
  Circle, 
  Clock, 
  AlertCircle,
  Trash2,
  Edit2
} from "lucide-react";
import { format } from "date-fns";

import { cn } from "@/utils";
import { 
  useTasks, 
  useCreateTask, 
  useUpdateTask, 
  useDeleteTask 
} from "@/hooks/use-query-example";
import { 
  Button 
} from "@/components/ui/button";
import { 
  Input 
} from "@/components/ui/input";
import { 
  Card, 
  CardContent, 
  CardHeader, 
  CardTitle, 
  CardDescription 
} from "@/components/ui/card";
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "@/components/ui/table";
import { 
  Badge 
} from "@/components/ui/badge";
import { 
  DropdownMenu, 
  DropdownMenuContent, 
  DropdownMenuItem, 
  DropdownMenuLabel, 
  DropdownMenuSeparator, 
  DropdownMenuTrigger 
} from "@/components/ui/dropdown-menu";
import { 
  Sheet, 
  SheetContent, 
  SheetDescription, 
  SheetHeader, 
  SheetTitle, 
  SheetTrigger,
  SheetFooter
} from "@/components/ui/sheet";
import { 
  Label 
} from "@/components/ui/label";
import { 
  Select, 
  SelectContent, 
  SelectItem, 
  SelectTrigger, 
  SelectValue 
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import type { Task, TaskCreateDto } from "@/services/task";

export default function TasksPage() {
  const [statusFilter, setStatusFilter] = useState<Task['status'] | 'all'>('all');
  const [searchTerm, setSearchTerm] = useState('');
  
  // Data Fetching
  const { data, isLoading } = useTasks(statusFilter === 'all' ? undefined : statusFilter);
  const tasks = data?.data || [];
  
  // Mutations
  const createTask = useCreateTask();
  const updateTask = useUpdateTask();
  const deleteTask = useDeleteTask();

  // Filter tasks locally by search term
  const filteredTasks = tasks.filter(task => 
    task.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
    task.description?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Tasks</h2>
          <p className="text-muted-foreground italic text-sm">
            Manage your project tasks with optimistic updates and global error handling.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <TaskForm 
            onSubmit={(data) => createTask.mutate(data)}
            isPending={createTask.isPending}
          />
        </div>
      </div>
      
      <Card className="border-border/60 bg-surface-1/50 backdrop-blur-sm shadow-sm transition-all hover:shadow-md">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div className="flex flex-1 items-center gap-2 max-w-md">
              <div className="relative w-full">
                <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  type="search"
                  placeholder="Search tasks..."
                  className="pl-8 bg-background/50 border-muted-foreground/20 focus-visible:ring-primary/30"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
              <Select 
                value={statusFilter} 
                onValueChange={(val) => setStatusFilter(val as any)}
              >
                <SelectTrigger className="w-[140px] bg-background/50 border-muted-foreground/20">
                  <SelectValue placeholder="All Status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Status</SelectItem>
                  <SelectItem value="todo">To Do</SelectItem>
                  <SelectItem value="doing">In Progress</SelectItem>
                  <SelectItem value="done">Done</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {[...Array(5)].map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent border-muted/50">
                  <TableHead className="w-[400px]">Task</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Priority</TableHead>
                  <TableHead>Created At</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredTasks.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="h-24 text-center text-muted-foreground italic">
                      No tasks found.
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredTasks.map((task) => (
                    <TableRow key={task.id} className="group border-muted/30 hover:bg-muted/5 transition-colors">
                      <TableCell className="font-medium">
                        <div className="flex flex-col gap-1">
                          <span className={task.status === 'done' ? 'line-through text-muted-foreground' : ''}>
                            {task.title}
                          </span>
                          {task.description && (
                            <span className="text-xs text-muted-foreground font-normal line-clamp-1">
                              {task.description}
                            </span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <StatusBadge status={task.status} />
                      </TableCell>
                      <TableCell>
                        <PriorityBadge priority={task.priority} />
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground font-normal">
                        {task.createdAt ? format(new Date(task.createdAt), 'MMM d, h:mm a') : '-'}
                      </TableCell>
                      <TableCell className="text-right">
                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity">
                              <MoreHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuLabel>Actions</DropdownMenuLabel>
                            <DropdownMenuSeparator />
                            <DropdownMenuItem 
                              onClick={() => updateTask.mutate({ 
                                id: task.id, 
                                data: { status: task.status === 'done' ? 'todo' : 'done' } 
                              })}
                            >
                              {task.status === 'done' ? 'Re-open' : 'Mark as Done'}
                            </DropdownMenuItem>
                            <TaskForm 
                              trigger={
                                <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
                                  Edit Details
                                </DropdownMenuItem>
                              }
                              task={task}
                              onSubmit={(data) => updateTask.mutate({ id: task.id, data })}
                              isPending={updateTask.isPending}
                            />
                            <DropdownMenuSeparator />
                            <DropdownMenuItem 
                              className="text-destructive focus:text-destructive focus:bg-destructive-subtle/30"
                              onClick={() => deleteTask.mutate(task.id)}
                            >
                              <Trash2 className="mr-2 h-4 w-4" />
                              Delete Task
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

// ============================================================================
// Helper Components
// ============================================================================

function StatusBadge({ status }: { status: Task['status'] }) {
  const configs = {
    todo: { label: 'To Do', icon: Circle, className: 'bg-muted/50 text-muted-foreground border-muted-foreground/30' },
    doing: { label: 'In Progress', icon: Clock, className: 'bg-info-subtle/50 text-info border-info/30' },
    done: { label: 'Done', icon: CheckCircle2, className: 'bg-success-subtle/50 text-success border-success/30' },
  };
  
  const config = configs[status];
  const Icon = config.icon;
  
  return (
    <Badge variant="outline" className={cn("gap-1 font-medium px-2 py-0.5", config.className)}>
      <Icon className="h-3 w-3" />
      {config.label}
    </Badge>
  );
}

function PriorityBadge({ priority }: { priority: Task['priority'] }) {
  const configs = {
    low: { label: 'Low', className: 'bg-muted/30 text-muted-foreground border-muted/50' },
    medium: { label: 'Medium', className: 'bg-warning-subtle/40 text-warning-strong border-warning/30' },
    high: { label: 'High', className: 'bg-destructive-subtle/40 text-destructive border-destructive/30 text-rose-600 dark:text-rose-400' },
  };
  
  const config = configs[priority || 'medium'];
  
  return (
    <Badge variant="outline" className={cn("text-[10px] uppercase tracking-wider font-bold h-5 px-1.5", config.className)}>
      {config.label}
    </Badge>
  );
}

function TaskForm({ 
  trigger, 
  task, 
  onSubmit, 
  isPending 
}: { 
  trigger?: React.ReactNode, 
  task?: Task, 
  onSubmit: (data: any) => void,
  isPending: boolean
}) {
  const [open, setOpen] = useState(false);
  const [formData, setFormData] = useState({
    title: task?.title || '',
    description: task?.description || '',
    status: task?.status || 'todo',
    priority: task?.priority || 'medium'
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(formData);
    setOpen(false);
  };

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        {trigger || (
          <Button className="bg-primary hover:bg-primary-strong shadow-sm hover:shadow-md transition-all gap-2">
            <Plus className="h-4 w-4" />
            Create Task
          </Button>
        )}
      </SheetTrigger>
      <SheetContent className="bg-surface-1/95 backdrop-blur-md border-l-border/60">
        <form onSubmit={handleSubmit} className="flex flex-col h-full">
          <SheetHeader>
            <SheetTitle>{task ? 'Edit Task' : 'Create New Task'}</SheetTitle>
            <SheetDescription>
              {task ? 'Modify the details of your existing task.' : 'Add a new task to your project tracking.'}
            </SheetDescription>
          </SheetHeader>
          <div className="flex-1 space-y-6 py-6 overflow-auto">
            <div className="space-y-2">
              <Label htmlFor="title" className="text-foreground/80 font-semibold">Title</Label>
              <Input 
                id="title" 
                value={formData.title} 
                onChange={e => setFormData(prev => ({ ...prev, title: e.target.value }))}
                placeholder="e.g. Implement Auth Guard"
                className="bg-background/40 focus-visible:ring-primary/20"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description" className="text-foreground/80 font-semibold">Description</Label>
              <textarea 
                id="description" 
                rows={4}
                value={formData.description} 
                onChange={e => setFormData(prev => ({ ...prev, description: e.target.value }))}
                placeholder="Detailed explanation of the task..."
                className="w-full flex min-h-[80px] rounded-md border border-input bg-background/40 px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/20 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="status" className="text-foreground/80 font-semibold">Status</Label>
                <Select value={formData.status} onValueChange={val => setFormData(prev => ({ ...prev, status: val as any }))}>
                  <SelectTrigger className="bg-background/40 border-muted-foreground/20">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="todo">To Do</SelectItem>
                    <SelectItem value="doing">In Progress</SelectItem>
                    <SelectItem value="done">Done</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="priority" className="text-foreground/80 font-semibold">Priority</Label>
                <Select value={formData.priority} onValueChange={val => setFormData(prev => ({ ...prev, priority: val as any }))}>
                  <SelectTrigger className="bg-background/40 border-muted-foreground/20">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="low">Low</SelectItem>
                    <SelectItem value="medium">Medium</SelectItem>
                    <SelectItem value="high">High</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>
          <SheetFooter className="pt-6 border-t border-border/40 mt-auto">
            <Button 
              type="submit" 
              className="w-full bg-primary hover:bg-primary-strong" 
              disabled={isPending}
            >
              {isPending ? 'Processing...' : (task ? 'Save Changes' : 'Create Task')}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}
