package events_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zgiai/luas/api/internal/infra/events"
)

// ============================================
// Scenario 1: Send a welcome email and initialize points after registration.
// ============================================

// UserCreatedEvent represents user creation.
type UserCreatedEvent struct {
	events.BaseEvent
	UserID   uint
	Username string
	Email    string
}

func (e UserCreatedEvent) EventName() string {
	return "user.created"
}

// EmailService sends user email.
type EmailService struct{}

func (s *EmailService) SendWelcomeEmail(ctx context.Context, email, username string) error {
	fmt.Printf("📧 Sending welcome email to %s (%s)\n", username, email)
	return nil
}

// PointsService manages user points.
type PointsService struct{}

func (s *PointsService) InitializePoints(ctx context.Context, userID uint) error {
	fmt.Printf("🎁 Initializing 100 points for user %d\n", userID)
	return nil
}

func ExampleEventBus_userRegistration() {
	bus := events.NewEventBus()
	emailSvc := &EmailService{}
	pointsSvc := &PointsService{}

	// Subscribe to user creation to send a welcome email.
	bus.Subscribe("user.created", func(ctx context.Context, event events.Event) error {
		e := event.(UserCreatedEvent)
		return emailSvc.SendWelcomeEmail(ctx, e.Email, e.Username)
	})

	// Subscribe to user creation to initialize points.
	bus.Subscribe("user.created", func(ctx context.Context, event events.Event) error {
		e := event.(UserCreatedEvent)
		return pointsSvc.InitializePoints(ctx, e.UserID)
	})

	// Simulate user registration.
	event := UserCreatedEvent{
		BaseEvent: events.NewBaseEventWithSource("user-service"),
		UserID:    1,
		Username:  "john",
		Email:     "john@example.com",
	}

	_ = bus.Publish(context.Background(), event)

	// Output:
	// 📧 Sending welcome email to john (john@example.com)
	// 🎁 Initializing 100 points for user 1
}

// ============================================
// Scenario 2: Observe order state changes with a wildcard subscription.
// ============================================

// OrderEvent is the base order event.
type OrderEvent struct {
	events.BaseEvent
	OrderID string
	Status  string
}

func (e OrderEvent) EventName() string {
	return "order." + e.Status
}

func ExampleEventBus_wildcardSubscription() {
	bus := events.NewEventBus()

	// Subscribe to every order event for logging.
	bus.Subscribe("order.*", func(ctx context.Context, event events.Event) error {
		e := event.(OrderEvent)
		fmt.Printf("📝 Order %s status changed to: %s\n", e.OrderID, e.Status)
		return nil
	})

	// Subscribe only to completed orders to send notifications.
	bus.Subscribe("order.completed", func(ctx context.Context, event events.Event) error {
		e := event.(OrderEvent)
		fmt.Printf("🎉 Order %s completed! Sending notification...\n", e.OrderID)
		return nil
	})

	// Publish order events with different states.
	ctx := context.Background()
	_ = bus.Publish(ctx, OrderEvent{BaseEvent: events.NewBaseEvent(), OrderID: "ORD-001", Status: "created"})
	_ = bus.Publish(ctx, OrderEvent{BaseEvent: events.NewBaseEvent(), OrderID: "ORD-001", Status: "paid"})
	_ = bus.Publish(ctx, OrderEvent{BaseEvent: events.NewBaseEvent(), OrderID: "ORD-001", Status: "completed"})

	// Output:
	// 📝 Order ORD-001 status changed to: created
	// 📝 Order ORD-001 status changed to: paid
	// 📝 Order ORD-001 status changed to: completed
	// 🎉 Order ORD-001 completed! Sending notification...
}

// ============================================
// Scenario 3: Check inventory before shipping through handler priorities.
// ============================================

func ExampleEventBus_priority() {
	bus := events.NewEventBus()

	// Low priority: process shipping.
	bus.Subscribe("order.paid", func(ctx context.Context, event events.Event) error {
		fmt.Println("3️⃣ Processing shipment...")
		return nil
	}, events.WithPriority(10))

	// High priority: check inventory first.
	bus.Subscribe("order.paid", func(ctx context.Context, event events.Event) error {
		fmt.Println("1️⃣ Checking inventory...")
		return nil
	}, events.WithPriority(100))

	// Medium priority: deduct inventory.
	bus.Subscribe("order.paid", func(ctx context.Context, event events.Event) error {
		fmt.Println("2️⃣ Deducting inventory...")
		return nil
	}, events.WithPriority(50))

	_ = bus.Publish(context.Background(), OrderEvent{
		BaseEvent: events.NewBaseEvent(),
		OrderID:   "ORD-002",
		Status:    "paid",
	})

	// Output:
	// 1️⃣ Checking inventory...
	// 2️⃣ Deducting inventory...
	// 3️⃣ Processing shipment...
}

// ============================================
// Scenario 4: Run asynchronous work without blocking the primary flow.
// ============================================

func ExampleEventBus_async() {
	bus := events.NewEventBus()
	var wg sync.WaitGroup
	wg.Add(1)

	// Send an SMS asynchronously without blocking the primary flow.
	bus.Subscribe("user.created", func(ctx context.Context, event events.Event) error {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Simulate a slow operation.
		fmt.Println("📱 SMS sent (async)")
		return nil
	}, events.WithAsync())

	// Update statistics synchronously.
	bus.Subscribe("user.created", func(ctx context.Context, event events.Event) error {
		fmt.Println("📊 Stats updated (sync)")
		return nil
	})

	event := UserCreatedEvent{
		BaseEvent: events.NewBaseEvent(),
		UserID:    2,
		Username:  "jane",
		Email:     "jane@example.com",
	}

	_ = bus.Publish(context.Background(), event)
	fmt.Println("✅ Publish returned immediately")

	wg.Wait() // Wait for asynchronous completion.

	// Output:
	// 📊 Stats updated (sync)
	// ✅ Publish returned immediately
	// 📱 SMS sent (async)
}

// ============================================
// Scenario 5: Compose logging and tracing middleware.
// ============================================

func ExampleEventBus_middleware() {
	bus := events.NewEventBus()

	// Logging middleware.
	bus.Use(func(next events.EventHandler) events.EventHandler {
		return func(ctx context.Context, event events.Event) error {
			fmt.Printf("🔍 [LOG] Event: %s, ID: %s\n", event.EventName(), event.Metadata().ID[:8])
			return next(ctx, event)
		}
	})

	// Duration tracing middleware.
	bus.Use(func(next events.EventHandler) events.EventHandler {
		return func(ctx context.Context, event events.Event) error {
			start := time.Now()
			err := next(ctx, event)
			fmt.Printf("⏱️ [TRACE] %s took %v\n", event.EventName(), time.Since(start))
			return err
		}
	})

	bus.Subscribe("payment.received", func(ctx context.Context, event events.Event) error {
		fmt.Println("💰 Processing payment...")
		return nil
	})

	_ = bus.Publish(context.Background(), OrderEvent{
		BaseEvent: events.NewBaseEvent(),
		OrderID:   "ORD-003",
		Status:    "received",
	})

	// Output shows middleware wrapping the handler
}

// ============================================
// Scenario 6: Preserve event correlation and causation.
// ============================================

func ExampleEventBus_correlation() {
	bus := events.NewEventBus()

	// Trigger a payment event when an order is created.
	bus.Subscribe("order.created", func(ctx context.Context, event events.Event) error {
		orderEvent := event.(OrderEvent)
		meta := event.Metadata()

		// Create the correlated payment event.
		paymentEvent := OrderEvent{
			BaseEvent: events.NewBaseEventWithCorrelation(
				meta.CorrelationID, // Preserve the correlation ID.
				meta.ID,            // Use the current event ID as the causation ID.
				"payment-service",
			),
			OrderID: orderEvent.OrderID,
			Status:  "payment_initiated",
		}

		fmt.Printf("📦 Order %s created (correlation: %s)\n", orderEvent.OrderID, meta.CorrelationID[:8])
		return bus.Publish(ctx, paymentEvent)
	})

	bus.Subscribe("order.payment_initiated", func(ctx context.Context, event events.Event) error {
		meta := event.Metadata()
		fmt.Printf("💳 Payment initiated (correlation: %s, caused by: %s)\n",
			meta.CorrelationID[:8], meta.CausationID[:8])
		return nil
	})

	// Create the initial event with a correlation ID.
	initialEvent := OrderEvent{
		BaseEvent: events.NewBaseEventWithCorrelation("corr-12345678", "", "order-service"),
		OrderID:   "ORD-004",
		Status:    "created",
	}

	_ = bus.Publish(context.Background(), initialEvent)

	// Output shows correlation tracking across events
}

// ============================================
// Scenario 7: Unsubscribe a handler.
// ============================================

func ExampleSubscription_unsubscribe() {
	bus := events.NewEventBus()

	callCount := 0
	sub := bus.Subscribe("order.test", func(ctx context.Context, event events.Event) error {
		callCount++
		fmt.Printf("Handler called %d time(s)\n", callCount)
		return nil
	})

	// Publish once while subscribed.
	_ = bus.Publish(context.Background(), OrderEvent{BaseEvent: events.NewBaseEvent(), OrderID: "1", Status: "test"})

	// Unsubscribe the handler.
	sub.Unsubscribe()
	fmt.Println("Unsubscribed")

	// Publish again; the handler must not run.
	_ = bus.Publish(context.Background(), OrderEvent{BaseEvent: events.NewBaseEvent(), OrderID: "2", Status: "test"})
	fmt.Printf("Final call count: %d\n", callCount)

	// Output:
	// Handler called 1 time(s)
	// Unsubscribed
	// Final call count: 1
}
