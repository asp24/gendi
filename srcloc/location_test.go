package srcloc

import (
	"testing"
)

func TestLocation_String(t *testing.T) {
	tests := []struct {
		name string
		loc  *Location
		want string
	}{
		{
			name: "full location",
			loc:  &Location{File: "/path/to/file.yaml", Line: 42, Column: 10},
			want: "/path/to/file.yaml:42:10",
		},
		{
			name: "nil location",
			loc:  nil,
			want: "",
		},
		{
			name: "zero values",
			loc:  &Location{File: "", Line: 0, Column: 0},
			want: ":0:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.loc.String()
			if got != tt.want {
				t.Errorf("Location.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
