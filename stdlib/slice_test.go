package stdlib

import "testing"

func TestMakeSlice(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := MakeSlice[int]()
		if len(result) != 0 {
			t.Errorf("MakeSlice() length = %d, want 0", len(result))
		}
		if result == nil {
			t.Error("MakeSlice() should return empty slice, not nil")
		}
	})

	t.Run("single_item", func(t *testing.T) {
		result := MakeSlice[string]("hello")
		if len(result) != 1 {
			t.Errorf("len(result) = %d, want 1", len(result))
		}
		if result[0] != "hello" {
			t.Errorf("result[0] = %q, want %q", result[0], "hello")
		}
	})

	t.Run("multiple_items", func(t *testing.T) {
		result := MakeSlice[int](1, 2, 3, 4, 5)
		if len(result) != 5 {
			t.Errorf("len(result) = %d, want 5", len(result))
		}
		for i, expected := range []int{1, 2, 3, 4, 5} {
			if result[i] != expected {
				t.Errorf("result[%d] = %d, want %d", i, result[i], expected)
			}
		}
	})

	t.Run("interface_types", func(t *testing.T) {
		// Using built-in interface to test generic behavior
		var items []any = MakeSlice[any]("hello", 42, true)
		if len(items) != 3 {
			t.Errorf("len(result) = %d, want 3", len(items))
		}
	})

	t.Run("pointer_types", func(t *testing.T) {
		type Service struct {
			ID int
		}

		s1 := &Service{ID: 1}
		s2 := &Service{ID: 2}
		result := MakeSlice[*Service](s1, s2)

		if len(result) != 2 {
			t.Errorf("len(result) = %d, want 2", len(result))
		}
		if result[0].ID != 1 {
			t.Errorf("result[0].ID = %d, want 1", result[0].ID)
		}
		if result[1].ID != 2 {
			t.Errorf("result[1].ID = %d, want 2", result[1].ID)
		}
	})
}
