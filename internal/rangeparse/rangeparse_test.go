package rangeparse

import (
	"reflect"
	"testing"
)

func TestParseUint32List(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		min     uint32
		max     uint32
		want    []uint32
		wantAll bool
		wantErr bool
	}{
		{
			name:    "all",
			input:   "all",
			min:     1,
			max:     10,
			wantAll: true,
		},
		{
			name:    "single values",
			input:   "3,1,2",
			min:     1,
			max:     10,
			want:    []uint32{1, 2, 3},
			wantAll: false,
		},
		{
			name:    "range with overlap",
			input:   "1-3,2,5",
			min:     1,
			max:     10,
			want:    []uint32{1, 2, 3, 5},
			wantAll: false,
		},
		{
			name:    "invalid token",
			input:   "1-a",
			min:     1,
			max:     10,
			wantErr: true,
		},
		{
			name:    "out of bound",
			input:   "11",
			min:     1,
			max:     10,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, all, err := ParseUint32List(tc.input, tc.min, tc.max)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if all != tc.wantAll {
				t.Fatalf("all flag mismatch: got=%v want=%v", all, tc.wantAll)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("values mismatch: got=%v want=%v", got, tc.want)
			}
		})
	}
}
