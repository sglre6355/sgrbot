package in_memory

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/sglre6355/sgrbot/internal/modules/music_player/domain"
)

func waitForCount(mu *sync.Mutex, counter *int, target int) bool {
	deadline := time.After(time.Second)
	for {
		mu.Lock()
		n := *counter
		mu.Unlock()
		if n >= target {
			return true
		}
		select {
		case <-deadline:
			return false
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestChannelEventBus_PublishAndSubscribe(t *testing.T) {
	type args struct {
		eventType    reflect.Type
		publishEvent domain.Event
	}
	type want struct {
		receivedCount int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "delivers matching event to subscriber",
			args: args{
				eventType:    reflect.TypeOf(domain.TrackStartedEvent{}),
				publishEvent: domain.TrackStartedEvent{PlayerStateID: domain.NewPlayerStateID()},
			},
			want: want{receivedCount: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := NewChannelEventBus(10)
			defer bus.Close()

			var mu sync.Mutex
			count := 0

			if err := bus.Subscribe(tt.args.eventType, func(_ context.Context, _ domain.Event) {
				mu.Lock()
				count++
				mu.Unlock()
			}); err != nil {
				t.Fatalf("Subscribe: %v", err)
			}

			bus.Publish(context.Background(), tt.args.publishEvent)

			if !waitForCount(&mu, &count, tt.want.receivedCount) {
				t.Fatal("timed out waiting for event delivery")
			}

			mu.Lock()
			defer mu.Unlock()
			if count != tt.want.receivedCount {
				t.Fatalf("received: got %d, want %d", count, tt.want.receivedCount)
			}
		})
	}
}

func TestChannelEventBus_SubscribeMultipleHandlers(t *testing.T) {
	type args struct {
		handlerCount int
		publishEvent domain.Event
	}
	type want struct {
		totalCalls int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "all handlers called for single event",
			args: args{handlerCount: 2, publishEvent: domain.PlaybackPausedEvent{}},
			want: want{totalCalls: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := NewChannelEventBus(10)
			defer bus.Close()

			var mu sync.Mutex
			count := 0

			for range tt.args.handlerCount {
				if err := bus.Subscribe(
					reflect.TypeOf(tt.args.publishEvent),
					func(_ context.Context, _ domain.Event) {
						mu.Lock()
						count++
						mu.Unlock()
					},
				); err != nil {
					t.Fatalf("Subscribe: %v", err)
				}
			}

			bus.Publish(context.Background(), tt.args.publishEvent)

			if !waitForCount(&mu, &count, tt.want.totalCalls) {
				t.Fatal("timed out waiting for handlers")
			}

			mu.Lock()
			defer mu.Unlock()
			if count != tt.want.totalCalls {
				t.Fatalf("handlers called: got %d, want %d", count, tt.want.totalCalls)
			}
		})
	}
}

func TestChannelEventBus_OnlyMatchingTypeDelivered(t *testing.T) {
	type args struct {
		subscribeType reflect.Type
		publishEvents []domain.Event
	}
	type want struct {
		receivedCount int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "non-matching events are not delivered",
			args: args{
				subscribeType: reflect.TypeOf(domain.TrackStartedEvent{}),
				publishEvents: []domain.Event{
					domain.PlaybackStoppedEvent{},
					domain.TrackStartedEvent{},
				},
			},
			want: want{receivedCount: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := NewChannelEventBus(10)
			defer bus.Close()

			var mu sync.Mutex
			count := 0

			if err := bus.Subscribe(tt.args.subscribeType, func(_ context.Context, _ domain.Event) {
				mu.Lock()
				count++
				mu.Unlock()
			}); err != nil {
				t.Fatalf("Subscribe: %v", err)
			}

			for _, event := range tt.args.publishEvents {
				bus.Publish(context.Background(), event)
			}

			if !waitForCount(&mu, &count, tt.want.receivedCount) {
				t.Fatal("timed out")
			}

			// Allow time for unexpected extra deliveries
			time.Sleep(50 * time.Millisecond)

			mu.Lock()
			defer mu.Unlock()
			if count != tt.want.receivedCount {
				t.Fatalf("received: got %d, want %d", count, tt.want.receivedCount)
			}
		})
	}
}

func TestChannelEventBus_CloseBehavior(t *testing.T) {
	type args struct {
		closeCount int
	}
	type want struct {
		noPanic bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "publish after close does not panic",
			args: args{closeCount: 1},
			want: want{noPanic: true},
		},
		{
			name: "double close does not panic",
			args: args{closeCount: 2},
			want: want{noPanic: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := NewChannelEventBus(10)

			for range tt.args.closeCount {
				bus.Close()
			}

			// Should not panic
			bus.Publish(context.Background(), domain.PlaybackStoppedEvent{})
		})
	}
}

func TestChannelEventBus_Subscribe_InvalidType(t *testing.T) {
	type args struct {
		eventType reflect.Type
	}
	type want struct {
		hasErr bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "non-Event type returns error",
			args: args{eventType: reflect.TypeOf("not-an-event")},
			want: want{hasErr: true},
		},
		{
			name: "valid Event type succeeds",
			args: args{eventType: reflect.TypeOf(domain.TrackStartedEvent{})},
			want: want{hasErr: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := NewChannelEventBus(10)
			defer bus.Close()

			err := bus.Subscribe(tt.args.eventType, func(_ context.Context, _ domain.Event) {})

			if (err != nil) != tt.want.hasErr {
				t.Fatalf("err: got %v, wantErr %v", err, tt.want.hasErr)
			}
		})
	}
}

func TestChannelEventBus_DefaultBufferSize(t *testing.T) {
	type args struct {
		bufferSize int
	}
	type want struct {
		noPanic bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "zero buffer uses default",
			args: args{bufferSize: 0},
			want: want{noPanic: true},
		},
		{
			name: "negative buffer uses default",
			args: args{bufferSize: -1},
			want: want{noPanic: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := NewChannelEventBus(tt.args.bufferSize)
			defer bus.Close()

			// Should not panic
			bus.Publish(context.Background(), domain.PlaybackStoppedEvent{})
		})
	}
}
