package domain

import "testing"

func TestLoopMode_String(t *testing.T) {
	tests := []struct {
		name string
		mode LoopMode
		want string
	}{
		{
			name: "LoopModeNone returns none",
			mode: LoopModeNone,
			want: "none",
		},
		{
			name: "LoopModeTrack returns track",
			mode: LoopModeTrack,
			want: "track",
		},
		{
			name: "LoopModeQueue returns queue",
			mode: LoopModeQueue,
			want: "queue",
		},
		{
			name: "unknown mode returns none",
			mode: LoopMode(99),
			want: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("LoopMode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoopMode_IotaValues(t *testing.T) {
	// Verify iota values are as expected
	if LoopModeNone != 0 {
		t.Errorf("LoopModeNone = %d, want 0", LoopModeNone)
	}
	if LoopModeTrack != 1 {
		t.Errorf("LoopModeTrack = %d, want 1", LoopModeTrack)
	}
	if LoopModeQueue != 2 {
		t.Errorf("LoopModeQueue = %d, want 2", LoopModeQueue)
	}
}
