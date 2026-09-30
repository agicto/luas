package events

import "time"

// BasicEvent is the minimal event a publisher provides. The bus wraps it into an Event that carries
// metadata for tracing and correlation.
type BasicEvent interface {
	// EventName returns the unique identifier for the event
	EventName() string
	// OccurredAt returns when the event happened
	OccurredAt() time.Time
}
