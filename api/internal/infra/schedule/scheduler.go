package schedule

import (
	"context"

	"github.com/zgiai/luas/api/internal/capabilities/workflow"
)

// Task represents a schedulable task.
type Task = workflow.ScheduledTask

// TaskFunc adapts a function into a schedulable task.
type TaskFunc = workflow.ScheduledTaskFunc

// Event represents a scheduled event.
type Event = workflow.Event

// Scheduler manages scheduled events.
type Scheduler = workflow.Scheduler

// NewEvent creates a new scheduled event.
func NewEvent(name string, task Task) *Event {
	return workflow.NewEvent(name, task)
}

// Call creates a new event with a function.
func Call(name string, fn func(ctx context.Context) error) *Event {
	return workflow.Call(name, fn)
}

// New creates a new scheduler.
func New() *Scheduler {
	return workflow.NewScheduler()
}

// Global returns the global scheduler instance.
func Global() *Scheduler {
	return workflow.GlobalScheduler()
}

// Register registers an event with the global scheduler.
func Register(event *Event) *Scheduler {
	return workflow.GlobalScheduler().Register(event)
}

// Schedule creates and registers a new event.
func Schedule(name string, fn func(ctx context.Context) error) *Event {
	return workflow.Schedule(name, fn)
}

// Run runs due events on the global scheduler.
func Run(ctx context.Context) {
	workflow.RunSchedule(ctx)
}

// Start starts the global scheduler.
func Start(ctx context.Context) {
	workflow.StartSchedule(ctx)
}

// Stop stops the global scheduler.
func Stop() {
	workflow.StopSchedule()
}
