package domain

import (
	"testing"
	"time"
)

func newTestTrack(id string) Track {
	return *ConstructTrack(
		TrackID(id), "Track "+id, "Author", time.Minute,
		"https://example.com/"+id, "", TrackSourceYouTube, false,
	)
}

func newTestEntry(id string, isAutoPlay bool) QueueEntry {
	return ConstructQueueEntry(
		newTestTrack(id), UserID("user1"), time.Now(), isAutoPlay,
	)
}

func TestParseQueueID(t *testing.T) {
	validID := NewQueueID()

	type args struct {
		id string
	}
	type want struct {
		queueID QueueID
		err     error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "valid UUID v7",
			args: args{id: string(validID)},
			want: want{queueID: validID, err: nil},
		},
		{
			name: "invalid string",
			args: args{id: "not-a-uuid"},
			want: want{queueID: "", err: ErrInvalidQueueID},
		},
		{
			name: "UUID v4 rejected",
			args: args{id: "550e8400-e29b-41d4-a716-446655440000"},
			want: want{queueID: "", err: ErrInvalidQueueID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseQueueID(tt.args.id)
			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if got != tt.want.queueID {
				t.Fatalf("id: got %q, want %q", got, tt.want.queueID)
			}
		})
	}
}

func TestQueueEntry_Accessors(t *testing.T) {
	type args struct {
		trackID     string
		requesterID UserID
		addedAt     time.Time
		isAutoPlay  bool
	}
	type want struct {
		trackID     TrackID
		requesterID UserID
		year        int
		isAutoPlay  bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "auto-play entry",
			args: args{
				trackID: "t1", requesterID: UserID("user42"),
				addedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), isAutoPlay: true,
			},
			want: want{
				trackID: TrackID("t1"), requesterID: UserID("user42"),
				year: 2025, isAutoPlay: true,
			},
		},
		{
			name: "manual entry",
			args: args{
				trackID: "t2", requesterID: UserID("user1"),
				addedAt: time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), isAutoPlay: false,
			},
			want: want{
				trackID: TrackID("t2"), requesterID: UserID("user1"),
				year: 2026, isAutoPlay: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := ConstructQueueEntry(
				newTestTrack(
					tt.args.trackID,
				),
				tt.args.requesterID,
				tt.args.addedAt,
				tt.args.isAutoPlay,
			)

			if entry.Track().ID() != tt.want.trackID {
				t.Errorf("Track ID: got %q, want %q", entry.Track().ID(), tt.want.trackID)
			}
			if entry.RequesterID() != tt.want.requesterID {
				t.Errorf("RequesterID: got %q, want %q", entry.RequesterID(), tt.want.requesterID)
			}
			if entry.AddedAt().Year() != tt.want.year {
				t.Errorf("AddedAt year: got %d, want %d", entry.AddedAt().Year(), tt.want.year)
			}
			if entry.IsAutoPlay() != tt.want.isAutoPlay {
				t.Errorf("IsAutoPlay: got %v, want %v", entry.IsAutoPlay(), tt.want.isAutoPlay)
			}
		})
	}
}

func TestNewQueueEntry(t *testing.T) {
	type args struct {
		trackID    string
		userID     UserID
		isAutoPlay bool
	}
	type want struct {
		isAutoPlay bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "sets addedAt to now and preserves fields",
			args: args{trackID: "t1", userID: UserID("user1"), isAutoPlay: false},
			want: want{isAutoPlay: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			entry := NewQueueEntry(
				newTestTrack(tt.args.trackID),
				tt.args.userID,
				tt.args.isAutoPlay,
			)
			after := time.Now()

			if entry.IsAutoPlay() != tt.want.isAutoPlay {
				t.Errorf("IsAutoPlay: got %v, want %v", entry.IsAutoPlay(), tt.want.isAutoPlay)
			}
			if entry.AddedAt().Before(before) || entry.AddedAt().After(after) {
				t.Error("AddedAt should be set to approximately time.Now()")
			}
		})
	}
}

func TestQueue_Operations(t *testing.T) {
	type args struct {
		setup func(t *testing.T) *Queue
	}
	type want struct {
		len     int
		isEmpty bool
		ids     []string // expected track IDs in order
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "new queue is empty",
			args: args{setup: func(_ *testing.T) *Queue {
				q := NewQueue()
				return &q
			}},
			want: want{len: 0, isEmpty: true, ids: nil},
		},
		{
			name: "append adds to end",
			args: args{setup: func(_ *testing.T) *Queue {
				q := NewQueue()
				q.Append(newTestEntry("1", false), newTestEntry("2", false))
				return &q
			}},
			want: want{len: 2, isEmpty: false, ids: []string{"1", "2"}},
		},
		{
			name: "prepend adds to front",
			args: args{setup: func(_ *testing.T) *Queue {
				q := NewQueue()
				q.Append(newTestEntry("1", false))
				q.Prepend(newTestEntry("0", false))
				return &q
			}},
			want: want{len: 2, isEmpty: false, ids: []string{"0", "1"}},
		},
		{
			name: "insert at index",
			args: args{setup: func(t *testing.T) *Queue {
				t.Helper()
				q := NewQueue()
				q.Append(newTestEntry("1", false), newTestEntry("3", false))
				if err := q.Insert(1, newTestEntry("2", false)); err != nil {
					t.Fatalf("setup Insert: %v", err)
				}
				return &q
			}},
			want: want{len: 3, isEmpty: false, ids: []string{"1", "2", "3"}},
		},
		{
			name: "remove at index",
			args: args{setup: func(t *testing.T) *Queue {
				t.Helper()
				q := NewQueue()
				q.Append(
					newTestEntry("1", false),
					newTestEntry("2", false),
					newTestEntry("3", false),
				)
				if _, err := q.Remove(1); err != nil {
					t.Fatalf("setup Remove: %v", err)
				}
				return &q
			}},
			want: want{len: 2, isEmpty: false, ids: []string{"1", "3"}},
		},
		{
			name: "clear empties queue",
			args: args{setup: func(_ *testing.T) *Queue {
				q := NewQueue()
				q.Append(newTestEntry("1", false), newTestEntry("2", false))
				q.Clear()
				return &q
			}},
			want: want{len: 0, isEmpty: true, ids: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.args.setup(t)

			if q.Len() != tt.want.len {
				t.Errorf("Len: got %d, want %d", q.Len(), tt.want.len)
			}
			if q.IsEmpty() != tt.want.isEmpty {
				t.Errorf("IsEmpty: got %v, want %v", q.IsEmpty(), tt.want.isEmpty)
			}
			for i, wantID := range tt.want.ids {
				got, err := q.Get(i)
				if err != nil {
					t.Fatalf("Get(%d): %v", i, err)
				}
				if string(got.Track().ID()) != wantID {
					t.Errorf("index %d: got %q, want %q", i, got.Track().ID(), wantID)
				}
			}
		})
	}
}

func TestQueue_Get_InvalidIndex(t *testing.T) {
	type args struct {
		setup func() *Queue
		index int
	}
	type want struct {
		err error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "empty queue",
			args: args{
				setup: func() *Queue { q := NewQueue(); return &q },
				index: 0,
			},
			want: want{err: ErrInvalidIndex},
		},
		{
			name: "negative index",
			args: args{
				setup: func() *Queue {
					q := NewQueue()
					q.Append(newTestEntry("1", false))
					return &q
				},
				index: -1,
			},
			want: want{err: ErrInvalidIndex},
		},
		{
			name: "out of bounds",
			args: args{
				setup: func() *Queue {
					q := NewQueue()
					q.Append(newTestEntry("1", false))
					return &q
				},
				index: 1,
			},
			want: want{err: ErrInvalidIndex},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.args.setup()
			_, err := q.Get(tt.args.index)
			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
		})
	}
}

func TestQueue_Insert_InvalidIndex(t *testing.T) {
	type args struct {
		index int
	}
	type want struct {
		err error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "insert into empty queue",
			args: args{index: 0},
			want: want{err: ErrInvalidIndex},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			err := q.Insert(tt.args.index, newTestEntry("1", false))
			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
		})
	}
}

func TestQueue_Remove_InvalidIndex(t *testing.T) {
	type args struct {
		index int
	}
	type want struct {
		err error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "remove from empty queue",
			args: args{index: 0},
			want: want{err: ErrInvalidIndex},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			_, err := q.Remove(tt.args.index)
			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
		})
	}
}

func TestQueue_List_ReturnsCopy(t *testing.T) {
	type args struct {
		entryIDs []string
	}
	type want struct {
		originalPreserved bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "mutating returned list does not affect queue",
			args: args{entryIDs: []string{"1"}},
			want: want{originalPreserved: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			for _, id := range tt.args.entryIDs {
				q.Append(newTestEntry(id, false))
			}

			list := q.List()
			list[0] = newTestEntry("modified", false)

			got, _ := q.Get(0)
			if (got.Track().ID() == TrackID("1")) != tt.want.originalPreserved {
				t.Error("List should return a copy, original queue was mutated")
			}
		})
	}
}

func TestQueue_FilterEntries(t *testing.T) {
	type args struct {
		entries []struct {
			id         string
			isAutoPlay bool
		}
	}
	type want struct {
		manualCount   int
		autoPlayCount int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "filters manual and auto-play entries",
			args: args{entries: []struct {
				id         string
				isAutoPlay bool
			}{
				{id: "manual1", isAutoPlay: false},
				{id: "auto1", isAutoPlay: true},
				{id: "manual2", isAutoPlay: false},
				{id: "auto2", isAutoPlay: true},
			}},
			want: want{manualCount: 2, autoPlayCount: 2},
		},
		{
			name: "all manual entries",
			args: args{entries: []struct {
				id         string
				isAutoPlay bool
			}{
				{id: "m1", isAutoPlay: false},
				{id: "m2", isAutoPlay: false},
			}},
			want: want{manualCount: 2, autoPlayCount: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			for _, e := range tt.args.entries {
				q.Append(newTestEntry(e.id, e.isAutoPlay))
			}

			if got := len(q.ManualPlayEntries()); got != tt.want.manualCount {
				t.Errorf("ManualPlayEntries: got %d, want %d", got, tt.want.manualCount)
			}
			if got := len(q.AutoPlayEntries()); got != tt.want.autoPlayCount {
				t.Errorf("AutoPlayEntries: got %d, want %d", got, tt.want.autoPlayCount)
			}
		})
	}
}

func TestQueue_Shuffle(t *testing.T) {
	type args struct {
		count int
	}
	type want struct {
		preservesLength bool
		changesOrder    bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "shuffles 20 entries",
			args: args{count: 20},
			want: want{preservesLength: true, changesOrder: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueue()
			for i := range tt.args.count {
				q.Append(newTestEntry(string(rune('A'+i)), false))
			}

			original := q.List()
			q.Shuffle()
			shuffled := q.List()

			if (len(shuffled) == len(original)) != tt.want.preservesLength {
				t.Errorf("length preserved: got %d vs %d", len(shuffled), len(original))
			}

			sameOrder := true
			for i := range original {
				if original[i].Track().ID() != shuffled[i].Track().ID() {
					sameOrder = false
					break
				}
			}
			if sameOrder == tt.want.changesOrder {
				t.Error("shuffle did not change order")
			}
		})
	}
}
