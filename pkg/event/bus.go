package event

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	// Channel buffer size for event subscriptions
	eventChannelBuffer = 100

	// Default timeout for publish operations
	defaultPublishTimeout = 5 * time.Second
)

// Bus is a publish-subscribe event bus.
// It allows multiple components to subscribe to events and publish events asynchronously.
type Bus struct {
	mu     sync.RWMutex
	subs   map[EventType][]*subscription
	logger *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type subscription struct {
	id        uint64
	ch        chan Event
	filter    func(Event) bool
	closeOnce sync.Once
}

// NewBus creates a new event bus.
func NewBus(logger *slog.Logger) *Bus {
	ctx, cancel := context.WithCancel(context.Background())
	return &Bus{
		subs:   make(map[EventType][]*subscription),
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Close stops the event bus and closes all subscriber channels.
func (b *Bus) Close() {
	b.cancel()
	b.wg.Wait()
}

// Publish sends an event to all matching subscribers.
// Returns an error if the context is cancelled.
func (b *Bus) Publish(ctx context.Context, evt Event) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Ensure event has timestamp
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}

	b.mu.RLock()
	receivers := make([]chan Event, 0)
	for eventType, subs := range b.subs {
		if eventType != "" && eventType != evt.Type {
			continue
		}

		for _, sub := range subs {
			// Apply filter if present
			if sub.filter != nil && !sub.filter(evt) {
				continue
			}

			receivers = append(receivers, sub.ch)
		}
	}
	b.mu.RUnlock()

	// Send to all receivers (non-blocking)
	sent := 0
	for _, ch := range receivers {
		select {
		case ch <- evt:
			sent++
		default:
			// Channel full, skip this subscriber
			b.logger.Warn("Event channel full, event dropped",
				"type", evt.Type,
				"subscribers", len(receivers),
				"sent", sent)
		}
	}

	if b.logger != nil && b.logger.Enabled(ctx, slog.LevelDebug) {
		b.logger.Debug("Event published",
			"type", evt.Type,
			"subscribers", len(receivers),
			"sent", sent,
			"dropped", len(receivers)-sent)
	}

	return nil
}

// Subscribe returns a channel for receiving events.
// The channel is buffered and will be closed when the bus is closed or Unsubscribe is called.
//
// The filter function is optional. If provided, only events matching the filter will be sent.
//
// Example:
//
//	ch, unsubscribe := bus.Subscribe(nil, EventDeviceStateChange)
//	defer unsubscribe()
//	for evt := range ch {
//	    stateChange := evt.Data.(DeviceStateChangeEvent)
//	    // Handle event
//	}
func (b *Bus) Subscribe(ctx context.Context, filter func(Event) bool) (<-chan Event, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, eventChannelBuffer)
	sub := &subscription{
		id:     uint64(time.Now().UnixNano()),
		ch:     ch,
		filter: filter,
	}

	// Determine which event types to subscribe to
	var eventTypes []EventType
	if filter == nil {
		// Subscribe to all event types (wildcard)
		eventTypes = []EventType{""}
	} else {
		// Don't know the filter's intent, subscribe to all for now
		// The filter will be applied during publish
		eventTypes = []EventType{""}
	}

	for _, eventType := range eventTypes {
		b.subs[eventType] = append(b.subs[eventType], sub)
	}

	b.wg.Add(1)

	unsubscribe := func() {
		sub.closeOnce.Do(func() {
			close(ch)

			b.mu.Lock()
			for eventType, subs := range b.subs {
				for i, s := range subs {
					if s.id == sub.id {
						b.subs[eventType] = append(subs[:i], subs[i+1:]...)
						break
					}
				}
			}
			b.mu.Unlock()

			b.wg.Done()
		})

		b.closeSubscriber(sub)
	}

	return ch, unsubscribe
}

// SubscribeTo returns a channel for receiving specific event types.
func (b *Bus) SubscribeTo(ctx context.Context, eventTypes ...EventType) (<-chan Event, func()) {
	if len(eventTypes) == 0 {
		return b.Subscribe(ctx, nil)
	}

	// Create filter that matches any of the specified event types
	filter := func(evt Event) bool {
		for _, et := range eventTypes {
			if evt.Type == et {
				return true
			}
		}
		return false
	}

	return b.Subscribe(ctx, filter)
}

// closeSubscriber handles cleanup for a subscription.
func (b *Bus) closeSubscriber(sub *subscription) {
	// Any additional cleanup can be done here
}

// SubscribersCount returns the number of active subscribers for each event type.
func (b *Bus) SubscribersCount() map[EventType]int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	counts := make(map[EventType]int)
	for eventType, subs := range b.subs {
		if eventType == "" {
			counts["*"] = len(subs)
		} else {
			counts[eventType] = len(subs)
		}
	}
	return counts
}

// PublishSync publishes an event synchronously, waiting for all subscribers to receive it.
// This is useful for critical events where you need to ensure delivery.
// Returns the number of subscribers that received the event.
func (b *Bus) PublishSync(ctx context.Context, evt Event) (int, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}

	b.mu.RLock()
	receivers := make([]*subscription, 0)
	for eventType, subs := range b.subs {
		if eventType != "" && eventType != evt.Type {
			continue
		}

		for _, sub := range subs {
			if sub.filter != nil && !sub.filter(evt) {
				continue
			}
			receivers = append(receivers, sub)
		}
	}
	b.mu.RUnlock()

	sent := 0
	for _, sub := range receivers {
		select {
		case sub.ch <- evt:
			sent++
		case <-ctx.Done():
			return sent, ctx.Err()
		case <-time.After(defaultPublishTimeout):
			b.logger.Warn("PublishSync timeout waiting for subscriber",
				"type", evt.Type)
		}
	}

	return sent, nil
}

// PublishDeviceJoined publishes a DeviceJoinedEvent.
func (b *Bus) PublishDeviceJoined(ctx context.Context, event DeviceJoinedEvent) error {
	return b.Publish(ctx, Event{
		Type: EventDeviceJoined,
		Data: event,
	})
}

// PublishDeviceLeft publishes a DeviceLeftEvent.
func (b *Bus) PublishDeviceLeft(ctx context.Context, event DeviceLeftEvent) error {
	return b.Publish(ctx, Event{
		Type: EventDeviceLeft,
		Data: event,
	})
}

// PublishDeviceStateChange publishes a DeviceStateChangeEvent.
func (b *Bus) PublishDeviceStateChange(ctx context.Context, event DeviceStateChangeEvent) error {
	if event.ChangedAt.IsZero() {
		event.ChangedAt = time.Now()
	}
	return b.Publish(ctx, Event{
		Type: EventDeviceStateChange,
		Data: event,
	})
}

// PublishSensorReport publishes a SensorReportEvent.
func (b *Bus) PublishSensorReport(ctx context.Context, event SensorReportEvent) error {
	if event.ReceivedAt.IsZero() {
		event.ReceivedAt = time.Now()
	}
	return b.Publish(ctx, Event{
		Type: EventSensorReport,
		Data: event,
	})
}

// PublishAdapterOnline publishes an AdapterOnlineEvent.
func (b *Bus) PublishAdapterOnline(ctx context.Context) error {
	return b.Publish(ctx, Event{
		Type: EventAdapterOnline,
		Data: AdapterOnlineEvent{
			OnlineAt: time.Now(),
		},
	})
}

// PublishAdapterOffline publishes an AdapterOfflineEvent.
func (b *Bus) PublishAdapterOffline(ctx context.Context, reason string) error {
	return b.Publish(ctx, Event{
		Type: EventAdapterOffline,
		Data: AdapterOfflineEvent{
			Reason:    reason,
			OfflineAt: time.Now(),
		},
	})
}
