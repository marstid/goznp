package event_test

import (
	"context"
	"testing"
	"time"

	"github.com/marstid/goznp/pkg/event"
)

func TestBusNew(t *testing.T) {
	bus := event.NewBus(nil)
	if bus == nil {
		t.Fatal("NewBus returned nil")
	}
}

func TestBusPublish(t *testing.T) {
	bus := event.NewBus(nil)
	defer bus.Close()

	ctx := context.Background()
	evt := event.Event{
		Type:      event.EventDeviceJoined,
		Timestamp: time.Now(),
		Data:      event.DeviceJoinedEvent{IEEEAddr: "test"},
	}

	err := bus.Publish(ctx, evt)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
}

func TestBusSubscribe(t *testing.T) {
	bus := event.NewBus(nil)
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, unsubscribe := bus.Subscribe(ctx, func(evt event.Event) bool {
		return evt.Type == event.EventDeviceJoined
	})
	defer unsubscribe()

	evt := event.Event{
		Type:      event.EventDeviceJoined,
		Timestamp: time.Now(),
		Data:      event.DeviceJoinedEvent{IEEEAddr: "1234"},
	}

	bus.Publish(ctx, evt)

	select {
	case received := <-ch:
		if received.Type != event.EventDeviceJoined {
			t.Errorf("Received wrong event type: got %v, want %v", received.Type, event.EventDeviceJoined)
		}
	case <-ctx.Done():
		t.Fatal("Timed out waiting for event")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}
}

func TestBusSubscribeTo(t *testing.T) {
	bus := event.NewBus(nil)
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, unsubscribe := bus.SubscribeTo(ctx, event.EventDeviceJoined, event.EventDeviceLeft)
	defer unsubscribe()

	evt := event.Event{
		Type:      event.EventDeviceJoined,
		Timestamp: time.Now(),
		Data:      event.DeviceJoinedEvent{IEEEAddr: "1234"},
	}
	bus.Publish(ctx, evt)

	select {
	case received := <-ch:
		if received.Type != event.EventDeviceJoined {
			t.Errorf("Received wrong event type: got %v", received.Type)
		}
	case <-ctx.Done():
		t.Fatal("Timed out waiting for event")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}

	evt2 := event.Event{
		Type:      event.EventDeviceLeft,
		Timestamp: time.Now(),
		Data:      event.DeviceLeftEvent{IEEEAddr: "5678"},
	}
	bus.Publish(ctx, evt2)

	select {
	case received := <-ch:
		if received.Type != event.EventDeviceLeft {
			t.Errorf("Received wrong event type: got %v", received.Type)
		}
	case <-ctx.Done():
		t.Fatal("Timed out waiting for event")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}
}

func TestBusFilter(t *testing.T) {
	bus := event.NewBus(nil)
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	receivedCount := 0

	ch, unsubscribe := bus.Subscribe(ctx, func(evt event.Event) bool {
		return evt.Type == "test"
	})
	defer unsubscribe()

	go func() {
		for {
			select {
			case <-ch:
				receivedCount++
			case <-ctx.Done():
				return
			}
		}
	}()

	bus.Publish(ctx, event.Event{
		Type: "test",
		Data: "data",
	})

	bus.Publish(ctx, event.Event{
		Type: event.EventDeviceJoined,
		Data: event.DeviceJoinedEvent{IEEEAddr: "1234"},
	})

	time.Sleep(50 * time.Millisecond)

	if receivedCount != 1 {
		t.Errorf("Expected 1 event, got %d", receivedCount)
	}
}

func TestBusSubscribersCount(t *testing.T) {
	bus := event.NewBus(nil)
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

func TestBusClose(t *testing.T) {
	bus := event.NewBus(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, unsubscribe := bus.Subscribe(ctx, nil)

	// Unsubscribe before Close to allow WaitGroup to complete
	unsubscribe()
	bus.Close()

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Expected channel to be closed after bus.Close()")
		}
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Timeout waiting for channel to close")
	}
}

func TestBusPublishDeviceJoined(t *testing.T) {
	bus := event.NewBus(nil)
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, unsubscribe := bus.Subscribe(ctx, nil)
	defer unsubscribe()

	evt := event.DeviceJoinedEvent{
		IEEEAddr: "00:11:22:33:44:55:66:77",
		NwkAddr:  "0x1234",
		Name:     "Test Device",
	}

	bus.PublishDeviceJoined(ctx, evt)

	select {
	case received := <-ch:
		if received.Type != event.EventDeviceJoined {
			t.Errorf("Wrong event type: got %v, want %v", received.Type, event.EventDeviceJoined)
		}
	case <-ctx.Done():
		t.Fatal("Timed out")
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout")
	}
}

func TestEventString(t *testing.T) {
	evt := event.Event{
		Type:      event.EventDeviceJoined,
		Timestamp: time.Now(),
		Data:      "test data",
	}

	str := evt.String()
	if str != string(event.EventDeviceJoined) {
		t.Errorf("Expected EventString to be type only, got %s", str)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := event.NewBus(nil)
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var sub1Received, sub2Received, sub3Received bool
	ch1, cancel1 := bus.Subscribe(ctx, nil)
	defer cancel1()

	ch2, cancel2 := bus.Subscribe(ctx, nil)
	defer cancel2()

	ch3, cancel3 := bus.Subscribe(ctx, nil)
	defer cancel3()

	go func() {
		<-ch1
		sub1Received = true
	}()

	go func() {
		<-ch2
		sub2Received = true
	}()

	go func() {
		<-ch3
		sub3Received = true
	}()

	evt := event.Event{
		Type: event.EventDeviceJoined,
		Data: event.DeviceJoinedEvent{IEEEAddr: "test"},
	}
	bus.Publish(ctx, evt)

	time.Sleep(50 * time.Millisecond)

	if !sub1Received || !sub2Received || !sub3Received {
		t.Errorf("Not all subscribers received event: sub1=%v, sub2=%v, sub3=%v",
			sub1Received, sub2Received, sub3Received)
	}
}
