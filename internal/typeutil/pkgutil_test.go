package typeutil

import (
	"reflect"
	"testing"
)

func TestSplitQualifiedName(t *testing.T) {
	tests := []struct {
		input   string
		wantPkg string
		wantSym string
		wantErr bool
	}{
		{"github.com/pkg.Symbol", "github.com/pkg", "Symbol", false},
		{"pkg.Symbol", "pkg", "Symbol", false},
		{"github.com/pkg.Symbol[int]", "github.com/pkg", "Symbol", false},
		{"github.com/pkg.Func[github.com/other.Type]", "github.com/pkg", "Func", false},
		{"Symbol", "", "", true},
		{".Symbol", "", "", true},
		{"pkg.", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg, sym, err := SplitQualifiedName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitQualifiedName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if pkg != tt.wantPkg {
				t.Errorf("SplitQualifiedName(%q) pkg = %q, want %q", tt.input, pkg, tt.wantPkg)
			}
			if sym != tt.wantSym {
				t.Errorf("SplitQualifiedName(%q) sym = %q, want %q", tt.input, sym, tt.wantSym)
			}
		})
	}
}

func TestSplitQualifiedNameWithTypeParams(t *testing.T) {
	tests := []struct {
		input      string
		wantPkg    string
		wantSym    string
		wantParams []string
		wantErr    bool
	}{
		{
			"github.com/pkg.Symbol",
			"github.com/pkg", "Symbol", nil, false,
		},
		{
			"github.com/pkg.Func[int]",
			"github.com/pkg", "Func", []string{"int"}, false,
		},
		{
			"github.com/pkg.Func[github.com/other.Type]",
			"github.com/pkg", "Func", []string{"github.com/other.Type"}, false,
		},
		{
			"github.com/pkg.Func[string, int]",
			"github.com/pkg", "Func", []string{"string", "int"}, false,
		},
		{
			"github.com/pkg.Func[map[string]int, github.com/other.Type]",
			"github.com/pkg", "Func", []string{"map[string]int", "github.com/other.Type"}, false,
		},
		{
			"github.com/pkg.Func[*github.com/other.Type]",
			"github.com/pkg", "Func", []string{"*github.com/other.Type"}, false,
		},
		{
			"github.com/pkg.Func[chan github.com/other.Event]",
			"github.com/pkg", "Func", []string{"chan github.com/other.Event"}, false,
		},
		{
			"github.com/pkg.Func[int",
			"", "", nil, true, // Missing closing bracket
		},
		{
			"Symbol",
			"", "", nil, true, // No package path
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			pkg, sym, params, err := SplitQualifiedNameWithTypeParams(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitQualifiedNameWithTypeParams(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if pkg != tt.wantPkg {
				t.Errorf("pkg = %q, want %q", pkg, tt.wantPkg)
			}
			if sym != tt.wantSym {
				t.Errorf("sym = %q, want %q", sym, tt.wantSym)
			}
			if !reflect.DeepEqual(params, tt.wantParams) {
				t.Errorf("params = %v, want %v", params, tt.wantParams)
			}
		})
	}
}

func TestSplitTypeParams(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"int", []string{"int"}},
		{"string, int", []string{"string", "int"}},
		{"map[string]int, Type", []string{"map[string]int", "Type"}},
		{"A, B[C, D], E", []string{"A", "B[C, D]", "E"}},
		{"", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitTypeParams(tt.input)
			if len(tt.want) == 0 && len(got) == 0 {
				return // Both empty, OK
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitTypeParams(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
