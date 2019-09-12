package glee_test

import (
	"testing"
)

func TestExecutor_Pkg005_Array(t *testing.T) {
	prog := MustBuildProgram(t, "./testdata/pkg005_array")

	t.Run("Slice", func(t *testing.T) {
		fn := MustFindFunction(t, prog, "arraySlice")
		e := NewExecutor(fn)
		defer e.Close()

		// Initial state should run until the 'if' statement.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `slice.go:12`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// Next state should execute the true 'if' block.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `slice.go:13`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if _, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if got, exp := string(values[0])[1:3], "XY"; got != exp {
			t.Fatalf("values[0]=%s, expected contains %s", got, exp)
		}

		// Next state should execute the false 'if' block.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `slice.go:15`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if _, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if got, exp := string(values[0])[1:3], "XY"; got == exp {
			t.Fatalf("values[0]=%s, expected NOT contains %s", got, exp)
		}
	})
}
