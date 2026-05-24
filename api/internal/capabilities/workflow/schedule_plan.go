package workflow

import (
	"time"

	"github.com/zgiai/zgo/internal/infra/schedule"
)

// SchedulePlan is the workflow-facing builder over scheduler events.
type SchedulePlan struct {
	manager *Manager
	event   *schedule.Event
}

// EveryMinute runs the task every minute.
func (p *SchedulePlan) EveryMinute() *SchedulePlan {
	p.event.EveryMinute()
	return p
}

// EveryFiveMinutes runs the task every five minutes.
func (p *SchedulePlan) EveryFiveMinutes() *SchedulePlan {
	p.event.EveryFiveMinutes()
	return p
}

// Hourly runs the task every hour.
func (p *SchedulePlan) Hourly() *SchedulePlan {
	p.event.Hourly()
	return p
}

// Daily runs the task every day at midnight.
func (p *SchedulePlan) Daily() *SchedulePlan {
	p.event.Daily()
	return p
}

// DailyAt runs the task every day at the specified time.
func (p *SchedulePlan) DailyAt(hour, minute int) *SchedulePlan {
	p.event.DailyAt(hour, minute)
	return p
}

// WeeklyOn runs the task weekly on the given weekday and time string (HH:MM).
func (p *SchedulePlan) WeeklyOn(day time.Weekday, timeStr string) *SchedulePlan {
	p.event.WeeklyOn(day, timeStr)
	return p
}

// WithoutOverlapping prevents concurrent executions of the same schedule.
func (p *SchedulePlan) WithoutOverlapping() *SchedulePlan {
	p.event.WithoutOverlapping()
	return p
}

// RunInBackground allows the schedule to execute asynchronously.
func (p *SchedulePlan) RunInBackground() *SchedulePlan {
	p.event.RunInBackground()
	return p
}

// Timezone applies a timezone to the schedule.
func (p *SchedulePlan) Timezone(name string) *SchedulePlan {
	p.event.Timezone(name)
	return p
}

// OnFailure registers a failure callback.
func (p *SchedulePlan) OnFailure(fn func(error)) *SchedulePlan {
	p.event.OnFailure(fn)
	return p
}

// OnSuccess registers a success callback.
func (p *SchedulePlan) OnSuccess(fn func()) *SchedulePlan {
	p.event.OnSuccess(fn)
	return p
}

// Register makes the schedule active on the manager's scheduler.
func (p *SchedulePlan) Register() *schedule.Event {
	p.manager.scheduler.Register(p.event)
	return p.event
}

// Event returns the underlying scheduled event for advanced use cases.
func (p *SchedulePlan) Event() *schedule.Event {
	return p.event
}
