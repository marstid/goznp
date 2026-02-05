package event

import (
	"context"
	"testing"
	"time"
)

func TestBusNew(t *testing.T) {
	bus := NewBus(nil)
	if bus == nil {
		t.Fatal("NewBus returned nil")
	}
}

func TestBusPublish(t *testing.T) {
	bus := NewBus(nil)
	defer bus.Close()

	ctx := context.Background()
	evt := Event{
		Type:      EventDeviceJoined,
		Timestamp: time.Now(),
		Data:      DeviceJoinedEvent{IEEEAddr: "test"},
	}

	err := bus.Publish(ctx, evt)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
}

func TestBusSubscribe(t *testing.T) {
	bus := NewBus(nil)
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, unsubscribe := bus.Subscribe(ctx, func(evt Event) bool {
		return evt.Type == EventDeviceJoined
	})
	defer unsubscribe()

	evt := Event{
		Type:      EventDeviceJoined,
		Timestamp: time.Now(),
		Data:      DeviceJoinedEvent{IEEEAddr: "1234"},
	}

	bus.Publish(ctx, evt)

	select {
	case received := <-ch:
		if received.Type != EventDeviceJoined {
			t.Errorf("Received wrong event type: got %v, want %v", received.Type, EventDeviceJoined)
		}
	case <-ctx.Done():
		t.Fatal("Timed out waiting for event")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}
}

func TestBusSubscribeTo(t *testing.T) {
	bus := NewBus(nil)
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, unsubscribe := bus.SubscribeTo(ctx, EventDeviceJoined, EventDeviceLeft)
	defer unsubscribe()

	evt := Event{
		Type:      EventDeviceJoined,
		Timestamp: time.Now(),
		Data:      DeviceJoinedEvent{IEEEAddr: "1234"},
	}
	bus.Publish(ctx, evt)

	select {
	case received := <-ch:
		if received.Type != EventDeviceJoined {
			t.Errorf("Received wrong event type: got %v", received.Type)
		}
	case <-ctx.Done():
		t.Fatal("Timed out waiting for event")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}

	evt2 := Event{
		Type:      EventDeviceLeft,
		Timestamp: time.Now(),
		Data:      DeviceLeftEvent{IEEEAddr: "5678"},
	}
	bus.Publish(ctx, evt2)

	select {
	case received := <-ch:
		if received.Type != EventDeviceLeft {
			t.Errorf("Received wrong event type: got %v", received.Type)
		}
	case <-ctx.Done():
		t.Fatal("Timed out waiting for event")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}
}

func TestBusSubscribersCount(t *testing.T) {
	bus := NewBus(nil)
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	counts := bus.SubscribersCount()
	if counts["*"] != 0 {
		t.Errorf("Expected 0 wildcard subscribers, got %d", counts["*"])
	}

	ch1, unsubscribe1 := bus.Subscribe(ctx, nil)
	_ = ch1

	counts = bus.SubscribersCount()
	if counts["*"] != 1 {
		t.Errorf("Expected 1 wildcard subscriber, got %d", counts["*"])
	}

	ch2, unsubscribe2 := bus.Subscribe(ctx, nil)
	_ = ch2

	counts = bus.SubscribersCount()
	if counts["*"] != 2 {
		t.Errorf("Expected 2 wildcard subscribers, got %d", counts["*"])
	}

	unsubscribe1()
	counts = bus.SubscribersCount()
	if counts["*"] != 1 {
		t.Errorf("Expected 1 wildcard subscriber after unsubscribe, got %d", counts["*"])
	}

	unsubscribe2()
}

func TestBusPublishDeviceJoined(t *testing.T) {
	bus := NewBus(nil)
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, unsubscribe := bus.Subscribe(ctx, nil)
	defer unsubscribe()

	evt := DeviceJoinedEvent{
		IEEEAddr: "00:11:22:33:44:55:66:77",
		NwkAddr:  "0x1234",
		Name:     "Test Device",
	}

	bus.PublishDeviceJoined(ctx, evt)

	select {
	case received := <-ch:
		if received.Type != EventDeviceJoined {
			t.Errorf("Wrong event type: got %v, want %v", received.Type, EventDeviceJoined)
		}
	case <-ctx.Done():
		t.Fatal("Timed out")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout")
	}
}

func TestEventString(t *testing.T) {
	evt := Event{
		Type:      EventDeviceJoined,
		Timestamp: time.Now(),
		Data:      "test data",
	}

	str := evt.String()
	if str != string(EventDeviceJoined) {
		t.Errorf("Expected EventString to be type only, got %s", str)
	}
}

func TestBusPublishSync(t *testing.T) {
	bus := NewBus(nil)
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	evt := Event{
		Type:      EventDeviceJoined,
		Timestamp: time.Now(),
		Data:      DeviceJoinedEvent{IEEEAddr: "test"},
	}

	sent, err := bus.PublishSync(ctx, evt)
	if err != nil {
		t.Fatalf("PublishSync failed: %v", err)
	}

	// No subscribers yet, so 0 events should be sent
	if sent != 0 {
		t.Errorf("Expected 0 events sent (no subscribers), got %d", sent)
	}

	// Add a subscriber
	ch, unsubscribe := bus.Subscribe(ctx, nil)
	defer unsubscribe()

	sent, err = bus.PublishSync(ctx, evt)
	if err != nil {
		t.Fatalf("PublishSync failed with subscriber: %v", err)
	}

	if sent != 1 {
		t.Errorf("Expected 1 event sent, got %d", sent)
	}

	// Verify the subscriber received it
	select {
	case <-ch:
		// Success
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Timeout waiting for synchronous delivery")
	}
}
