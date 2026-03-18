package domain

import (
	"testing"
)

func TestParsePlayerStateID(t *testing.T) {
	validID := NewPlayerStateID()

	type args struct {
		id string
	}
	type want struct {
		playerStateID PlayerStateID
		err           error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "valid UUID v7",
			args: args{id: string(validID)},
			want: want{playerStateID: validID, err: nil},
		},
		{
			name: "invalid string",
			args: args{id: "not-valid"},
			want: want{playerStateID: "", err: ErrInvalidPlayerStateID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePlayerStateID(tt.args.id)
			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if got != tt.want.playerStateID {
				t.Fatalf("id: got %q, want %q", got, tt.want.playerStateID)
			}
		})
	}
}

func TestParseLoopMode(t *testing.T) {
	type args struct {
		s string
	}
	type want struct {
		mode LoopMode
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "none", args: args{s: "none"}, want: want{mode: LoopModeNone}},
		{name: "track", args: args{s: "track"}, want: want{mode: LoopModeTrack}},
		{name: "queue", args: args{s: "queue"}, want: want{mode: LoopModeQueue}},
		{
			name: "invalid defaults to none",
			args: args{s: "invalid"},
			want: want{mode: LoopModeNone},
		},
		{name: "empty defaults to none", args: args{s: ""}, want: want{mode: LoopModeNone}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseLoopMode(tt.args.s); got != tt.want.mode {
				t.Fatalf("got %v, want %v", got, tt.want.mode)
			}
		})
	}
}

func TestLoopMode_String(t *testing.T) {
	type args struct {
		mode LoopMode
	}
	type want struct {
		str string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "none", args: args{mode: LoopModeNone}, want: want{str: "none"}},
		{name: "track", args: args{mode: LoopModeTrack}, want: want{str: "track"}},
		{name: "queue", args: args{mode: LoopModeQueue}, want: want{str: "queue"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.mode.String(); got != tt.want.str {
				t.Fatalf("got %q, want %q", got, tt.want.str)
			}
		})
	}
}

func TestPlayerState_Append(t *testing.T) {
	type args struct {
		initialIDs []string
		appendID   string
	}
	type want struct {
		startIndex   int
		becameActive bool
		isActive     bool
		currentID    string
		currentIndex int
		queueLen     int
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
				startIndex: 0, becameActive: true, isActive: true,
				currentID: "1", currentIndex: 0, queueLen: 1,
			},
		},
		{
			name: "second append does not reactivate",
			args: args{initialIDs: []string{"1"}, appendID: "2"},
			want: want{
				startIndex: 1, becameActive: false, isActive: true,
				currentID: "1", currentIndex: 0, queueLen: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			for _, id := range tt.args.initialIDs {
				ps.Append(newTestEntry(id, false))
			}

			startIndex, becameActive := ps.Append(newTestEntry(tt.args.appendID, false))

			if startIndex != tt.want.startIndex {
				t.Errorf("startIndex: got %d, want %d", startIndex, tt.want.startIndex)
			}
			if becameActive != tt.want.becameActive {
				t.Errorf("becameActive: got %v, want %v", becameActive, tt.want.becameActive)
			}
			if ps.IsPlaybackActive() != tt.want.isActive {
				t.Errorf(
					"IsPlaybackActive: got %v, want %v",
					ps.IsPlaybackActive(),
					tt.want.isActive,
				)
			}
			if ps.Current().Track().ID() != TrackID(tt.want.currentID) {
				t.Errorf("Current: got %q, want %q", ps.Current().Track().ID(), tt.want.currentID)
			}
			if ps.CurrentIndex() != tt.want.currentIndex {
				t.Errorf("CurrentIndex: got %d, want %d", ps.CurrentIndex(), tt.want.currentIndex)
			}
			if ps.Len() != tt.want.queueLen {
				t.Errorf("Len: got %d, want %d", ps.Len(), tt.want.queueLen)
			}
		})
	}
}

func TestPlayerState_Prepend(t *testing.T) {
	type args struct {
		prependID string
	}
	type want struct {
		currentIndex int
		currentID    string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "adjusts current index",
			args: args{prependID: "0"},
			want: want{currentIndex: 1, currentID: "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			ps.Append(newTestEntry("1", false))

			ps.Prepend(newTestEntry(tt.args.prependID, false))

			if ps.CurrentIndex() != tt.want.currentIndex {
				t.Errorf("CurrentIndex: got %d, want %d", ps.CurrentIndex(), tt.want.currentIndex)
			}
			if ps.Current().Track().ID() != TrackID(tt.want.currentID) {
				t.Errorf("Current: got %q, want %q", ps.Current().Track().ID(), tt.want.currentID)
			}
		})
	}
}

func TestPlayerState_Insert(t *testing.T) {
	type args struct {
		initialIDs []string
		seekIndex  int
		insertAt   int
		insertID   string
	}
	type want struct {
		currentIndex int
		currentID    string
		err          error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "insert at current adjusts index",
			args: args{initialIDs: []string{"1", "2"}, seekIndex: 0, insertAt: 0, insertID: "0"},
			want: want{currentIndex: 1, currentID: "1", err: nil},
		},
		{
			name: "insert after current no adjustment",
			args: args{initialIDs: []string{"1", "3"}, seekIndex: 0, insertAt: 1, insertID: "2"},
			want: want{currentIndex: 0, currentID: "1", err: nil},
		},
		{
			name: "invalid index",
			args: args{initialIDs: []string{"1"}, seekIndex: 0, insertAt: 99, insertID: "x"},
			want: want{currentIndex: 0, currentID: "1", err: ErrInvalidIndex},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			for _, id := range tt.args.initialIDs {
				ps.Append(newTestEntry(id, false))
			}

			err := ps.Insert(tt.args.insertAt, newTestEntry(tt.args.insertID, false))

			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if err == nil {
				if ps.CurrentIndex() != tt.want.currentIndex {
					t.Errorf(
						"CurrentIndex: got %d, want %d",
						ps.CurrentIndex(),
						tt.want.currentIndex,
					)
				}
				if ps.Current().Track().ID() != TrackID(tt.want.currentID) {
					t.Errorf(
						"Current: got %q, want %q",
						ps.Current().Track().ID(),
						tt.want.currentID,
					)
				}
			}
		})
	}
}

func TestPlayerState_Seek(t *testing.T) {
	type args struct {
		index int
	}
	type want struct {
		isNil        bool
		trackID      string
		currentIndex int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "valid index",
			args: args{index: 2},
			want: want{isNil: false, trackID: "3", currentIndex: 2},
		},
		{
			name: "invalid index returns nil",
			args: args{index: 5},
			want: want{isNil: true, trackID: "", currentIndex: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			ps.Append(newTestEntry("1", false), newTestEntry("2", false), newTestEntry("3", false))

			entry := ps.Seek(tt.args.index)

			if tt.want.isNil {
				if entry != nil {
					t.Error("expected nil entry")
				}
			} else {
				if entry == nil {
					t.Fatal("expected non-nil entry")
				}
				if string(entry.Track().ID()) != tt.want.trackID {
					t.Errorf("track: got %q, want %q", entry.Track().ID(), tt.want.trackID)
				}
				if ps.CurrentIndex() != tt.want.currentIndex {
					t.Errorf(
						"CurrentIndex: got %d, want %d",
						ps.CurrentIndex(),
						tt.want.currentIndex,
					)
				}
			}
		})
	}
}

func TestPlayerState_Advance(t *testing.T) {
	type args struct {
		trackIDs  []string
		seekIndex int
		mode      LoopMode
	}
	type want struct {
		isNil        bool
		trackID      string
		currentIndex int
		isActive     bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "LoopModeNone advances to next",
			args: args{trackIDs: []string{"1", "2"}, seekIndex: 0, mode: LoopModeNone},
			want: want{isNil: false, trackID: "2", currentIndex: 1, isActive: true},
		},
		{
			name: "LoopModeNone at last returns nil",
			args: args{trackIDs: []string{"1", "2"}, seekIndex: 1, mode: LoopModeNone},
			want: want{isNil: true, trackID: "", currentIndex: 1, isActive: false},
		},
		{
			name: "LoopModeTrack repeats same track",
			args: args{trackIDs: []string{"1", "2"}, seekIndex: 0, mode: LoopModeTrack},
			want: want{isNil: false, trackID: "1", currentIndex: 0, isActive: true},
		},
		{
			name: "LoopModeQueue wraps to start",
			args: args{trackIDs: []string{"1", "2"}, seekIndex: 1, mode: LoopModeQueue},
			want: want{isNil: false, trackID: "1", currentIndex: 0, isActive: true},
		},
		{
			name: "empty queue returns nil",
			args: args{trackIDs: nil, seekIndex: 0, mode: LoopModeNone},
			want: want{isNil: true, trackID: "", currentIndex: 0, isActive: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			for _, id := range tt.args.trackIDs {
				ps.Append(newTestEntry(id, false))
			}
			if tt.args.seekIndex > 0 && len(tt.args.trackIDs) > 0 {
				ps.Seek(tt.args.seekIndex)
			}

			next := ps.Advance(tt.args.mode)

			if tt.want.isNil {
				if next != nil {
					t.Error("expected nil")
				}
			} else {
				if next == nil {
					t.Fatal("expected non-nil")
				}
				if string(next.Track().ID()) != tt.want.trackID {
					t.Errorf("track: got %q, want %q", next.Track().ID(), tt.want.trackID)
				}
			}
			if ps.IsPlaybackActive() != tt.want.isActive {
				t.Errorf(
					"IsPlaybackActive: got %v, want %v",
					ps.IsPlaybackActive(),
					tt.want.isActive,
				)
			}
		})
	}
}

func TestPlayerState_Played_Current_Upcoming(t *testing.T) {
	type args struct {
		trackIDs  []string
		seekIndex int
	}
	type want struct {
		playedIDs   []string
		currentID   string
		upcomingIDs []string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "seek to middle",
			args: args{trackIDs: []string{"1", "2", "3"}, seekIndex: 1},
			want: want{playedIDs: []string{"1"}, currentID: "2", upcomingIDs: []string{"3"}},
		},
		{
			name: "at first track",
			args: args{trackIDs: []string{"1", "2", "3"}, seekIndex: 0},
			want: want{playedIDs: nil, currentID: "1", upcomingIDs: []string{"2", "3"}},
		},
		{
			name: "at last track",
			args: args{trackIDs: []string{"1", "2", "3"}, seekIndex: 2},
			want: want{playedIDs: []string{"1", "2"}, currentID: "3", upcomingIDs: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			for _, id := range tt.args.trackIDs {
				ps.Append(newTestEntry(id, false))
			}
			ps.Seek(tt.args.seekIndex)

			played := ps.Played()
			if len(played) != len(tt.want.playedIDs) {
				t.Fatalf("Played len: got %d, want %d", len(played), len(tt.want.playedIDs))
			}
			for i, wantID := range tt.want.playedIDs {
				if string(played[i].Track().ID()) != wantID {
					t.Errorf("Played[%d]: got %q, want %q", i, played[i].Track().ID(), wantID)
				}
			}

			current := ps.Current()
			if current == nil || string(current.Track().ID()) != tt.want.currentID {
				t.Errorf("Current: got %v, want %q", current, tt.want.currentID)
			}

			upcoming := ps.Upcoming()
			if len(upcoming) != len(tt.want.upcomingIDs) {
				t.Fatalf("Upcoming len: got %d, want %d", len(upcoming), len(tt.want.upcomingIDs))
			}
			for i, wantID := range tt.want.upcomingIDs {
				if string(upcoming[i].Track().ID()) != wantID {
					t.Errorf("Upcoming[%d]: got %q, want %q", i, upcoming[i].Track().ID(), wantID)
				}
			}
		})
	}
}

func TestPlayerState_Current_WhenInactive(t *testing.T) {
	type args struct{}
	type want struct {
		playedIDs     []string
		currentIsNil  bool
		upcomingEmpty bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "inactive state returns nil current and empty upcoming",
			want: want{
				playedIDs:     []string{"1", "2", "3"},
				currentIsNil:  true,
				upcomingEmpty: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			ps.Append(newTestEntry("1", false))
			ps.Append(newTestEntry("2", false))
			ps.Append(newTestEntry("3", false))
			ps.Seek(2)
			ps.Advance(LoopModeNone)

			played := ps.Played()
			if len(played) != len(tt.want.playedIDs) {
				t.Fatalf("Played len: got %d, want %d", len(played), len(tt.want.playedIDs))
			}
			for i, wantID := range tt.want.playedIDs {
				if string(played[i].Track().ID()) != wantID {
					t.Errorf("Played[%d]: got %q, want %q", i, played[i].Track().ID(), wantID)
				}
			}

			if (ps.Current() == nil) != tt.want.currentIsNil {
				t.Errorf("Current nil: got %v, want %v", ps.Current() == nil, tt.want.currentIsNil)
			}
			if (len(ps.Upcoming()) == 0) != tt.want.upcomingEmpty {
				t.Errorf(
					"Upcoming empty: got %v, want %v",
					len(ps.Upcoming()) == 0,
					tt.want.upcomingEmpty,
				)
			}
		})
	}
}

func TestPlayerState_Remove(t *testing.T) {
	type args struct {
		trackIDs  []string
		seekIndex int
		removeAt  int
	}
	type want struct {
		removedID    string
		isActive     bool
		isEmpty      bool
		currentID    string
		currentIndex int
		err          error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "remove current track with next",
			args: args{trackIDs: []string{"1", "2"}, seekIndex: 0, removeAt: 0},
			want: want{
				removedID:    "1",
				isActive:     true,
				isEmpty:      false,
				currentID:    "2",
				currentIndex: 0,
				err:          nil,
			},
		},
		{
			name: "remove only track",
			args: args{trackIDs: []string{"1"}, seekIndex: 0, removeAt: 0},
			want: want{
				removedID:    "1",
				isActive:     false,
				isEmpty:      true,
				currentID:    "",
				currentIndex: 0,
				err:          nil,
			},
		},
		{
			name: "remove before current adjusts index",
			args: args{trackIDs: []string{"1", "2", "3"}, seekIndex: 2, removeAt: 0},
			want: want{
				removedID:    "1",
				isActive:     true,
				isEmpty:      false,
				currentID:    "3",
				currentIndex: 1,
				err:          nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			for _, id := range tt.args.trackIDs {
				ps.Append(newTestEntry(id, false))
			}
			if tt.args.seekIndex > 0 {
				ps.Seek(tt.args.seekIndex)
			}

			removed, err := ps.Remove(tt.args.removeAt)

			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if string(removed.Track().ID()) != tt.want.removedID {
				t.Errorf("removed: got %q, want %q", removed.Track().ID(), tt.want.removedID)
			}
			if ps.IsPlaybackActive() != tt.want.isActive {
				t.Errorf(
					"IsPlaybackActive: got %v, want %v",
					ps.IsPlaybackActive(),
					tt.want.isActive,
				)
			}
			if ps.IsEmpty() != tt.want.isEmpty {
				t.Errorf("IsEmpty: got %v, want %v", ps.IsEmpty(), tt.want.isEmpty)
			}
			if tt.want.isActive {
				if ps.CurrentIndex() != tt.want.currentIndex {
					t.Errorf(
						"CurrentIndex: got %d, want %d",
						ps.CurrentIndex(),
						tt.want.currentIndex,
					)
				}
				if string(ps.Current().Track().ID()) != tt.want.currentID {
					t.Errorf(
						"Current: got %q, want %q",
						ps.Current().Track().ID(),
						tt.want.currentID,
					)
				}
			}
		})
	}
}

func TestPlayerState_Clear(t *testing.T) {
	type args struct {
		trackIDs []string
	}
	type want struct {
		isActive     bool
		isEmpty      bool
		currentIndex int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "clears active state",
			args: args{trackIDs: []string{"1", "2"}},
			want: want{isActive: false, isEmpty: true, currentIndex: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			for _, id := range tt.args.trackIDs {
				ps.Append(newTestEntry(id, false))
			}

			ps.Clear()

			if ps.IsPlaybackActive() != tt.want.isActive {
				t.Errorf(
					"IsPlaybackActive: got %v, want %v",
					ps.IsPlaybackActive(),
					tt.want.isActive,
				)
			}
			if ps.IsEmpty() != tt.want.isEmpty {
				t.Errorf("IsEmpty: got %v, want %v", ps.IsEmpty(), tt.want.isEmpty)
			}
			if ps.CurrentIndex() != tt.want.currentIndex {
				t.Errorf("CurrentIndex: got %d, want %d", ps.CurrentIndex(), tt.want.currentIndex)
			}
		})
	}
}

func TestPlayerState_ClearExceptCurrent(t *testing.T) {
	type args struct {
		trackIDs  []string
		seekIndex int
		isActive  bool
	}
	type want struct {
		count        int
		currentID    string
		currentIndex int
		queueLen     int
		err          error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "clears all except current",
			args: args{trackIDs: []string{"1", "2", "3"}, seekIndex: 1, isActive: true},
			want: want{count: 2, currentID: "2", currentIndex: 0, queueLen: 1, err: nil},
		},
		{
			name: "not playing returns error",
			args: args{trackIDs: nil, seekIndex: 0, isActive: false},
			want: want{err: ErrNotPlaying},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			for _, id := range tt.args.trackIDs {
				ps.Append(newTestEntry(id, false))
			}
			if tt.args.seekIndex > 0 && tt.args.isActive {
				ps.Seek(tt.args.seekIndex)
			}

			count, err := ps.ClearExceptCurrent()

			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if err == nil {
				if count != tt.want.count {
					t.Errorf("count: got %d, want %d", count, tt.want.count)
				}
				if ps.Len() != tt.want.queueLen {
					t.Errorf("Len: got %d, want %d", ps.Len(), tt.want.queueLen)
				}
				if string(ps.Current().Track().ID()) != tt.want.currentID {
					t.Errorf(
						"Current: got %q, want %q",
						ps.Current().Track().ID(),
						tt.want.currentID,
					)
				}
				if ps.CurrentIndex() != tt.want.currentIndex {
					t.Errorf(
						"CurrentIndex: got %d, want %d",
						ps.CurrentIndex(),
						tt.want.currentIndex,
					)
				}
			}
		})
	}
}

func TestPlayerState_Pause(t *testing.T) {
	type args struct {
		isActive bool
		isPaused bool
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
			name: "pause active playback",
			args: args{isActive: true, isPaused: false},
			want: want{err: nil},
		},
		{
			name: "pause when not playing",
			args: args{isActive: false, isPaused: false},
			want: want{err: ErrNotPlaying},
		},
		{
			name: "pause when already paused",
			args: args{isActive: true, isPaused: true},
			want: want{err: ErrAlreadyPaused},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			if tt.args.isActive {
				ps.Append(newTestEntry("1", false))
			}
			if tt.args.isPaused {
				if err := ps.Pause(); err != nil {
					t.Fatalf("setup Pause: %v", err)
				}
			}

			err := ps.Pause()

			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if err == nil && !ps.IsPaused() {
				t.Error("should be paused")
			}
		})
	}
}

func TestPlayerState_Resume(t *testing.T) {
	type args struct {
		isActive bool
		isPaused bool
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
			name: "resume paused playback",
			args: args{isActive: true, isPaused: true},
			want: want{err: nil},
		},
		{
			name: "resume when not playing",
			args: args{isActive: false, isPaused: false},
			want: want{err: ErrNotPlaying},
		},
		{
			name: "resume when not paused",
			args: args{isActive: true, isPaused: false},
			want: want{err: ErrNotPaused},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			if tt.args.isActive {
				ps.Append(newTestEntry("1", false))
			}
			if tt.args.isPaused {
				if err := ps.Pause(); err != nil {
					t.Fatalf("setup Pause: %v", err)
				}
			}

			err := ps.Resume()

			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if err == nil && ps.IsPaused() {
				t.Error("should not be paused")
			}
		})
	}
}

func TestPlayerState_Skip(t *testing.T) {
	type args struct {
		trackIDs []string
		loopMode LoopMode
	}
	type want struct {
		skippedID string
		hasNext   bool
		nextID    string
		isActive  bool
		err       error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "skip to next track",
			args: args{trackIDs: []string{"1", "2"}, loopMode: LoopModeNone},
			want: want{skippedID: "1", hasNext: true, nextID: "2", isActive: true, err: nil},
		},
		{
			name: "skip last track",
			args: args{trackIDs: []string{"1"}, loopMode: LoopModeNone},
			want: want{skippedID: "1", hasNext: false, isActive: false, err: nil},
		},
		{
			name: "skip not playing",
			args: args{trackIDs: nil, loopMode: LoopModeNone},
			want: want{err: ErrNotPlaying},
		},
		{
			name: "skip overrides LoopModeTrack",
			args: args{trackIDs: []string{"1", "2"}, loopMode: LoopModeTrack},
			want: want{skippedID: "1", hasNext: true, nextID: "2", isActive: true, err: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			for _, id := range tt.args.trackIDs {
				ps.Append(newTestEntry(id, false))
			}
			ps.SetLoopMode(tt.args.loopMode)

			skipped, next, err := ps.Skip()

			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if err != nil {
				return
			}
			if string(skipped.Track().ID()) != tt.want.skippedID {
				t.Errorf("skipped: got %q, want %q", skipped.Track().ID(), tt.want.skippedID)
			}
			if tt.want.hasNext {
				if next == nil || string(next.Track().ID()) != tt.want.nextID {
					t.Errorf("next: got %v, want %q", next, tt.want.nextID)
				}
			} else if next != nil {
				t.Error("next should be nil")
			}
			if ps.IsPlaybackActive() != tt.want.isActive {
				t.Errorf(
					"IsPlaybackActive: got %v, want %v",
					ps.IsPlaybackActive(),
					tt.want.isActive,
				)
			}
		})
	}
}

func TestPlayerState_CycleLoopMode(t *testing.T) {
	type args struct {
		initialMode LoopMode
	}
	type want struct {
		nextMode LoopMode
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "none to track",
			args: args{initialMode: LoopModeNone},
			want: want{nextMode: LoopModeTrack},
		},
		{
			name: "track to queue",
			args: args{initialMode: LoopModeTrack},
			want: want{nextMode: LoopModeQueue},
		},
		{
			name: "queue to none",
			args: args{initialMode: LoopModeQueue},
			want: want{nextMode: LoopModeNone},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			ps.SetLoopMode(tt.args.initialMode)

			got := ps.CycleLoopMode()

			if got != tt.want.nextMode {
				t.Fatalf("got %v, want %v", got, tt.want.nextMode)
			}
		})
	}
}

func TestPlayerState_SetAutoPlayEnabled(t *testing.T) {
	type args struct {
		enabled bool
	}
	type want struct {
		enabled bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{name: "disable", args: args{enabled: false}, want: want{enabled: false}},
		{name: "enable", args: args{enabled: true}, want: want{enabled: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()

			ps.SetAutoPlayEnabled(tt.args.enabled)

			if ps.IsAutoPlayEnabled() != tt.want.enabled {
				t.Fatalf("got %v, want %v", ps.IsAutoPlayEnabled(), tt.want.enabled)
			}
		})
	}
}

func TestPlayerState_HasNext(t *testing.T) {
	type args struct {
		autoPlayEnabled bool
		trackIDs        []string
		mode            LoopMode
	}
	type want struct {
		hasNext bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "auto-play enabled always returns true",
			args: args{autoPlayEnabled: true, trackIDs: nil, mode: LoopModeNone},
			want: want{hasNext: true},
		},
		{
			name: "empty queue returns false",
			args: args{autoPlayEnabled: false, trackIDs: nil, mode: LoopModeNone},
			want: want{hasNext: false},
		},
		{
			name: "LoopModeTrack always returns true",
			args: args{autoPlayEnabled: false, trackIDs: []string{"1"}, mode: LoopModeTrack},
			want: want{hasNext: true},
		},
		{
			name: "LoopModeQueue always returns true",
			args: args{autoPlayEnabled: false, trackIDs: []string{"1"}, mode: LoopModeQueue},
			want: want{hasNext: true},
		},
		{
			name: "LoopModeNone at last returns false",
			args: args{autoPlayEnabled: false, trackIDs: []string{"1"}, mode: LoopModeNone},
			want: want{hasNext: false},
		},
		{
			name: "LoopModeNone not at last returns true",
			args: args{autoPlayEnabled: false, trackIDs: []string{"1", "2"}, mode: LoopModeNone},
			want: want{hasNext: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			ps.SetAutoPlayEnabled(tt.args.autoPlayEnabled)
			for _, id := range tt.args.trackIDs {
				ps.Append(newTestEntry(id, false))
			}

			if got := ps.HasNext(tt.args.mode); got != tt.want.hasNext {
				t.Fatalf("got %v, want %v", got, tt.want.hasNext)
			}
		})
	}
}

func TestPlayerState_IsAtLast(t *testing.T) {
	type args struct {
		trackIDs  []string
		seekIndex int
	}
	type want struct {
		isAtLast bool
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "not at last",
			args: args{trackIDs: []string{"1", "2"}, seekIndex: 0},
			want: want{isAtLast: false},
		},
		{
			name: "at last",
			args: args{trackIDs: []string{"1", "2"}, seekIndex: 1},
			want: want{isAtLast: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			for _, id := range tt.args.trackIDs {
				ps.Append(newTestEntry(id, false))
			}
			if tt.args.seekIndex > 0 {
				ps.Seek(tt.args.seekIndex)
			}

			if got := ps.IsAtLast(); got != tt.want.isAtLast {
				t.Fatalf("got %v, want %v", got, tt.want.isAtLast)
			}
		})
	}
}

func TestPlayerState_Shuffle_KeepsCurrentTrack(t *testing.T) {
	type args struct {
		count     int
		seekIndex int
	}
	type want struct {
		currentIndex int
		currentID    string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "current track moves to index 0",
			args: args{count: 20, seekIndex: 5},
			want: want{currentIndex: 0, currentID: string(rune('A' + 5))},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()
			for i := range tt.args.count {
				ps.Append(newTestEntry(string(rune('A'+i)), false))
			}
			ps.Seek(tt.args.seekIndex)

			ps.Shuffle()

			if ps.CurrentIndex() != tt.want.currentIndex {
				t.Errorf("CurrentIndex: got %d, want %d", ps.CurrentIndex(), tt.want.currentIndex)
			}
			if string(ps.Current().Track().ID()) != tt.want.currentID {
				t.Errorf("Current: got %q, want %q", ps.Current().Track().ID(), tt.want.currentID)
			}
		})
	}
}

func TestNewPlayerState_Defaults(t *testing.T) {
	type args struct{}
	type want struct {
		isActive        bool
		isPaused        bool
		autoPlayEnabled bool
		loopMode        LoopMode
		isEmpty         bool
		currentIndex    int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "default values",
			want: want{
				isActive: false, isPaused: false, autoPlayEnabled: true,
				loopMode: LoopModeNone, isEmpty: true, currentIndex: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPlayerState()

			if ps.ID() == "" {
				t.Error("ID should not be empty")
			}
			if ps.IsPlaybackActive() != tt.want.isActive {
				t.Errorf(
					"IsPlaybackActive: got %v, want %v",
					ps.IsPlaybackActive(),
					tt.want.isActive,
				)
			}
			if ps.IsPaused() != tt.want.isPaused {
				t.Errorf("IsPaused: got %v, want %v", ps.IsPaused(), tt.want.isPaused)
			}
			if ps.IsAutoPlayEnabled() != tt.want.autoPlayEnabled {
				t.Errorf(
					"IsAutoPlayEnabled: got %v, want %v",
					ps.IsAutoPlayEnabled(),
					tt.want.autoPlayEnabled,
				)
			}
			if ps.LoopMode() != tt.want.loopMode {
				t.Errorf("LoopMode: got %v, want %v", ps.LoopMode(), tt.want.loopMode)
			}
			if ps.IsEmpty() != tt.want.isEmpty {
				t.Errorf("IsEmpty: got %v, want %v", ps.IsEmpty(), tt.want.isEmpty)
			}
			if ps.CurrentIndex() != tt.want.currentIndex {
				t.Errorf("CurrentIndex: got %d, want %d", ps.CurrentIndex(), tt.want.currentIndex)
			}
		})
	}
}
