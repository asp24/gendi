package cmd

import "testing"

func TestStringSetFlag_String(t *testing.T) {
	cases := []struct {
		name string
		m    map[string]struct{}
		want string
	}{
		{"nil map", nil, ""},
		{"multiple values sorted", map[string]struct{}{"beta": {}, "alpha": {}, "gamma": {}}, "alpha,beta,gamma"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &stringSetFlag{values: &tc.m}
			if got := f.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStringSetFlag_Set(t *testing.T) {
	var m map[string]struct{}
	f := &stringSetFlag{values: &m}
	f.Set("a")
	f.Set("b")
	f.Set("a") // duplicate
	want := map[string]struct{}{"a": {}, "b": {}}
	if got := len(*f.values); got != len(want) {
		t.Fatalf("len = %d, want %d", got, len(want))
	}
	for k := range want {
		if _, ok := (*f.values)[k]; !ok {
			t.Errorf("missing key %q", k)
		}
	}
}
