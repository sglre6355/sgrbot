package domain

import "testing"

func TestParseUserID(t *testing.T) {
	type args struct {
		id string
	}
	type want struct {
		userID UserID
		err    error
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "valid user ID",
			args: args{id: "12345"},
			want: want{userID: UserID("12345"), err: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUserID(tt.args.id)
			if err != tt.want.err {
				t.Fatalf("err: got %v, want %v", err, tt.want.err)
			}
			if got != tt.want.userID {
				t.Fatalf("got %q, want %q", got, tt.want.userID)
			}
		})
	}
}

func TestUserID_String(t *testing.T) {
	type args struct {
		id UserID
	}
	type want struct {
		str string
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "returns underlying string",
			args: args{id: UserID("abc")},
			want: want{str: "abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.id.String(); got != tt.want.str {
				t.Fatalf("got %q, want %q", got, tt.want.str)
			}
		})
	}
}
