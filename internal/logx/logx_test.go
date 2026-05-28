package logx

import "testing"

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Level
		wantErr bool
	}{
		{name: "debug", input: "debug", want: LevelDebug},
		{name: "info default", input: "", want: LevelInfo},
		{name: "warn alias", input: "warning", want: LevelWarn},
		{name: "error", input: "error", want: LevelError},
		{name: "invalid", input: "trace", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseLevel(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("level mismatch: got=%v want=%v", got, tc.want)
			}
		})
	}
}
