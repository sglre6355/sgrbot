package domain

import (
	"context"
	"testing"
)

// stubRecommender implements TrackRecommender for testing.
type stubRecommender struct {
	tracks     []Track
	err        error
	acceptSeed bool
}

func (r *stubRecommender) GetRecommendation(
	_ context.Context,
	_ []QueueEntry,
	_ []QueueEntry,
	_ int,
) ([]Track, error) {
	return r.tracks, r.err
}

func (r *stubRecommender) AcceptsSeed(_ QueueEntry) bool {
	return r.acceptSeed
}

func newPlayerService() *PlayerService {
	return NewPlayerService(nil)
}

func newActiveState(ids ...string) *PlayerState {
	ps := NewPlayerState()
	for _, id := range ids {
		ps.Append(newTestEntry(id, false))
	}
	return ps
}

func assertEventType[T Event](t *testing.T, events []Event, index int) T {
	t.Helper()
	if index >= len(events) {
		t.Fatalf("expected event at index %d, but only %d events", index, len(events))
	}
	ev, ok := events[index].(T)
	if !ok {
		t.Fatalf("event[%d]: got %T, want %T", index, events[index], *new(T))
	}
	return ev
}

func TestPlayerService_Prepend(t *testing.T) {
	type args struct {
		initialIDs []string
		prependID  string
	}
	type want struct {
		queueLen   int
		eventCount int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "prepend emits TrackAddedEvent",
			args: args{initialIDs: []string{"1"}, prependID: "0"},
			want: want{queueLen: 2, eventCount: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPlayerService()
			ps := newActiveState(tt.args.initialIDs...)

			events := svc.Prepend(ps, newTestEntry(tt.args.prependID, false))

			if ps.Len() != tt.want.queueLen {
				t.Errorf("Len: got %d, want %d", ps.Len(), tt.want.queueLen)
			}
			if len(events) != tt.want.eventCount {
				t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
			}
			assertEventType[TrackAddedEvent](t, events, 0)
		})
	}
}

func TestPlayerService_Append(t *testing.T) {
	type args struct {
		initialIDs []string
		appendID   string
	}
	type want struct {
		startIndex   int
		becameActive bool
		eventCount   int
		eventTypes   []string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "first append activates playback",
			args: args{initialIDs: nil, appendID: "1"},
			want: want{
				startIndex:   0,
				becameActive: true,
				eventCount:   2,
				eventTypes:   []string{"TrackAdded", "TrackStarted"},
			},
		},
		{
			name: "append to active state",
			args: args{initialIDs: []string{"1"}, appendID: "2"},
			want: want{
				startIndex:   1,
				becameActive: false,
				eventCount:   1,
				eventTypes:   []string{"TrackAdded"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPlayerService()
			var ps *PlayerState
			if tt.args.initialIDs == nil {
				ps = NewPlayerState()
			} else {
				ps = newActiveState(tt.args.initialIDs...)
			}

			startIndex, becameActive, events := svc.Append(
				ps,
				newTestEntry(tt.args.appendID, false),
			)

			if startIndex != tt.want.startIndex {
				t.Errorf("startIndex: got %d, want %d", startIndex, tt.want.startIndex)
			}
			if becameActive != tt.want.becameActive {
				t.Errorf("becameActive: got %v, want %v", becameActive, tt.want.becameActive)
			}
			if len(events) != tt.want.eventCount {
				t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
			}
			assertEventType[TrackAddedEvent](t, events, 0)
			if tt.want.becameActive {
				assertEventType[TrackStartedEvent](t, events, 1)
			}
		})
	}
}

func TestPlayerService_Insert(t *testing.T) {
	type args struct {
		initialIDs []string
		index      int
		insertID   string
	}
	type want struct {
		eventCount int
		err        error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "valid insert emits TrackAddedEvent",
			args: args{initialIDs: []string{"1", "3"}, index: 1, insertID: "2"},
			want: want{eventCount: 1, err: nil},
		},
		{
			name: "invalid index returns error",
			args: args{initialIDs: []string{"1"}, index: 99, insertID: "2"},
			want: want{eventCount: 0, err: ErrInvalidIndex},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPlayerService()
			ps := newActiveState(tt.args.initialIDs...)

			events, err := svc.Insert(ps, tt.args.index, newTestEntry(tt.args.insertID, false))

			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if len(events) != tt.want.eventCount {
				t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
			}
			if err == nil {
				assertEventType[TrackAddedEvent](t, events, 0)
			}
		})
	}
}

func TestPlayerService_Seek(t *testing.T) {
	type args struct {
		initialIDs []string
		index      int
	}
	type want struct {
		isNil      bool
		trackID    string
		eventCount int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "valid seek emits TrackStartedEvent",
			args: args{initialIDs: []string{"1", "2", "3"}, index: 2},
			want: want{isNil: false, trackID: "3", eventCount: 1},
		},
		{
			name: "invalid index returns nil",
			args: args{initialIDs: []string{"1"}, index: 99},
			want: want{isNil: true, eventCount: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPlayerService()
			ps := newActiveState(tt.args.initialIDs...)

			entry, events := svc.Seek(ps, tt.args.index)

			if tt.want.isNil {
				if entry != nil {
					t.Error("entry should be nil")
				}
				if events != nil {
					t.Error("events should be nil")
				}
			} else {
				if entry == nil || string(entry.Track().ID()) != tt.want.trackID {
					t.Errorf("entry: got %v, want %q", entry, tt.want.trackID)
				}
				if len(events) != tt.want.eventCount {
					t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
				}
				assertEventType[TrackStartedEvent](t, events, 0)
			}
		})
	}
}

func TestPlayerService_Shuffle(t *testing.T) {
	type args struct {
		initialIDs []string
	}
	type want struct {
		eventCount int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "non-empty emits QueueShuffledEvent",
			args: args{initialIDs: []string{"1", "2", "3"}},
			want: want{eventCount: 1},
		},
		{
			name: "empty returns no events",
			args: args{initialIDs: nil},
			want: want{eventCount: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPlayerService()
			var ps *PlayerState
			if tt.args.initialIDs == nil {
				ps = NewPlayerState()
			} else {
				ps = newActiveState(tt.args.initialIDs...)
			}

			events := svc.Shuffle(ps)

			if len(events) != tt.want.eventCount {
				t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
			}
			if tt.want.eventCount > 0 {
				assertEventType[QueueShuffledEvent](t, events, 0)
			}
		})
	}
}

func TestPlayerService_Remove(t *testing.T) {
	type args struct {
		initialIDs []string
		index      int
	}
	type want struct {
		removedID  string
		eventCount int
		event0Type string
		event1Type string
		err        error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "remove non-current emits TrackRemovedEvent",
			args: args{initialIDs: []string{"1", "2", "3"}, index: 2},
			want: want{removedID: "3", eventCount: 1, event0Type: "removed", err: nil},
		},
		{
			name: "remove current with next emits TrackRemoved and TrackStarted",
			args: args{initialIDs: []string{"1", "2"}, index: 0},
			want: want{
				removedID:  "1",
				eventCount: 2,
				event0Type: "removed",
				event1Type: "started",
				err:        nil,
			},
		},
		{
			name: "remove only track emits TrackRemoved and PlaybackStopped",
			args: args{initialIDs: []string{"1"}, index: 0},
			want: want{
				removedID:  "1",
				eventCount: 2,
				event0Type: "removed",
				event1Type: "stopped",
				err:        nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPlayerService()
			ps := newActiveState(tt.args.initialIDs...)

			entry, events, err := svc.Remove(ps, tt.args.index)

			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if string(entry.Track().ID()) != tt.want.removedID {
				t.Errorf("removed: got %q, want %q", entry.Track().ID(), tt.want.removedID)
			}
			if len(events) != tt.want.eventCount {
				t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
			}
			assertEventType[TrackRemovedEvent](t, events, 0)
			switch tt.want.event1Type {
			case "started":
				assertEventType[TrackStartedEvent](t, events, 1)
			case "stopped":
				assertEventType[PlaybackStoppedEvent](t, events, 1)
			}
		})
	}
}

func TestPlayerService_Clear(t *testing.T) {
	type args struct {
		initialIDs []string
	}
	type want struct {
		count      int
		eventCount int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "clear active emits QueueCleared and PlaybackStopped",
			args: args{initialIDs: []string{"1", "2"}},
			want: want{count: 2, eventCount: 2},
		},
		{
			name: "clear empty emits no events",
			args: args{initialIDs: nil},
			want: want{count: 0, eventCount: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPlayerService()
			var ps *PlayerState
			if tt.args.initialIDs == nil {
				ps = NewPlayerState()
			} else {
				ps = newActiveState(tt.args.initialIDs...)
			}

			count, events := svc.Clear(ps)

			if count != tt.want.count {
				t.Errorf("count: got %d, want %d", count, tt.want.count)
			}
			if len(events) != tt.want.eventCount {
				t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
			}
			if tt.want.eventCount >= 2 {
				assertEventType[QueueClearedEvent](t, events, 0)
				assertEventType[PlaybackStoppedEvent](t, events, 1)
			}
		})
	}
}

func TestPlayerService_ClearExceptCurrent(t *testing.T) {
	type args struct {
		initialIDs []string
		isActive   bool
	}
	type want struct {
		count      int
		eventCount int
		err        error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "clears all except current",
			args: args{initialIDs: []string{"1", "2", "3"}, isActive: true},
			want: want{count: 2, eventCount: 1, err: nil},
		},
		{
			name: "not playing returns error",
			args: args{initialIDs: nil, isActive: false},
			want: want{err: ErrNotPlaying},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPlayerService()
			var ps *PlayerState
			if tt.args.isActive {
				ps = newActiveState(tt.args.initialIDs...)
			} else {
				ps = NewPlayerState()
			}

			count, events, err := svc.ClearExceptCurrent(ps)

			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if err == nil {
				if count != tt.want.count {
					t.Errorf("count: got %d, want %d", count, tt.want.count)
				}
				if len(events) != tt.want.eventCount {
					t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
				}
				assertEventType[QueueClearedEvent](t, events, 0)
			}
		})
	}
}

func TestPlayerService_Pause(t *testing.T) {
	type args struct {
		initialIDs []string
	}
	type want struct {
		eventCount int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "pause emits PlaybackPausedEvent",
			args: args{initialIDs: []string{"1"}},
			want: want{eventCount: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPlayerService()
			ps := newActiveState(tt.args.initialIDs...)

			events, err := svc.Pause(ps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events) != tt.want.eventCount {
				t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
			}
			assertEventType[PlaybackPausedEvent](t, events, 0)
		})
	}
}

func TestPlayerService_Resume(t *testing.T) {
	type args struct {
		initialIDs []string
	}
	type want struct {
		eventCount int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "resume emits PlaybackResumedEvent",
			args: args{initialIDs: []string{"1"}},
			want: want{eventCount: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPlayerService()
			ps := newActiveState(tt.args.initialIDs...)
			if err := ps.Pause(); err != nil {
				t.Fatalf("setup Pause: %v", err)
			}

			events, err := svc.Resume(ps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events) != tt.want.eventCount {
				t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
			}
			assertEventType[PlaybackResumedEvent](t, events, 0)
		})
	}
}

func TestPlayerService_Skip(t *testing.T) {
	type args struct {
		initialIDs []string
	}
	type want struct {
		skippedID  string
		eventCount int
		eventType  string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "skip with next emits TrackStartedEvent",
			args: args{initialIDs: []string{"1", "2"}},
			want: want{skippedID: "1", eventCount: 1, eventType: "started"},
		},
		{
			name: "skip last emits QueueExhaustedEvent",
			args: args{initialIDs: []string{"1"}},
			want: want{skippedID: "1", eventCount: 1, eventType: "exhausted"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newPlayerService()
			ps := newActiveState(tt.args.initialIDs...)

			skipped, events, err := svc.Skip(ps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(skipped.Track().ID()) != tt.want.skippedID {
				t.Errorf("skipped: got %q, want %q", skipped.Track().ID(), tt.want.skippedID)
			}
			if len(events) != tt.want.eventCount {
				t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
			}
			if tt.want.eventType == "started" {
				assertEventType[TrackStartedEvent](t, events, 0)
			} else {
				assertEventType[QueueExhaustedEvent](t, events, 0)
			}
		})
	}
}

func TestPlayerService_TryAutoPlay(t *testing.T) {
	type args struct {
		autoPlayEnabled bool
		autoPlayService *AutoPlayService
	}
	type want struct {
		hasEntry   bool
		trackID    string
		eventCount int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "disabled returns nil",
			args: args{autoPlayEnabled: false, autoPlayService: nil},
			want: want{hasEntry: false, eventCount: 0},
		},
		{
			name: "nil service returns nil",
			args: args{autoPlayEnabled: true, autoPlayService: nil},
			want: want{hasEntry: false, eventCount: 0},
		},
		{
			name: "success emits TrackAdded and TrackStarted",
			args: args{
				autoPlayEnabled: true,
				autoPlayService: NewAutoPlayService(UserID("bot"), &stubRecommender{
					tracks:     []Track{newTestTrack("rec1")},
					acceptSeed: true,
				}),
			},
			want: want{hasEntry: true, trackID: "rec1", eventCount: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewPlayerService(tt.args.autoPlayService)
			ps := newActiveState("1")
			ps.SetAutoPlayEnabled(tt.args.autoPlayEnabled)

			entry, events := svc.TryAutoPlay(context.Background(), ps)

			if tt.want.hasEntry {
				if entry == nil {
					t.Fatal("expected non-nil entry")
				}
				if string(entry.Track().ID()) != tt.want.trackID {
					t.Errorf("track: got %q, want %q", entry.Track().ID(), tt.want.trackID)
				}
			} else if entry != nil {
				t.Error("expected nil entry")
			}
			if len(events) != tt.want.eventCount {
				t.Fatalf("events: got %d, want %d", len(events), tt.want.eventCount)
			}
			if tt.want.eventCount >= 2 {
				assertEventType[TrackAddedEvent](t, events, 0)
				assertEventType[TrackStartedEvent](t, events, 1)
			}
		})
	}
}
