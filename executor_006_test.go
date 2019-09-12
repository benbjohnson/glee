package glee_test

import (
	"testing"

	"github.com/benbjohnson/glee"
)

func TestExecutor_Pkg006_Interface(t *testing.T) {
	prog := MustBuildProgram(t, "./testdata/pkg006_interface")

	t.Run("ChangeInterface", func(t *testing.T) {
		fn := MustFindFunction(t, prog, "changeInterface")
		e := NewExecutor(fn)
		defer e.Close()

		// Initial state should run until the 'if' statement.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `interface.go:21`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// After returning it should end on the  the 'if' statement.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `interface.go:12`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// Next state should execute the true 'if' block.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `interface.go:13`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if arrays, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if x, err := EvalVar(state, arrays, values, fn, "x"); err != nil {
			t.Fatal(err)
		} else if x.Value != 90 {
			t.Fatalf("unexpected 'x': %d", x.Value)
		}

		// Next state should execute the false 'if' block.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `interface.go:15`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if arrays, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if x, err := EvalVar(state, arrays, values, fn, "x"); err != nil {
			t.Fatal(err)
		} else if x.Value == 90 {
			t.Fatalf("unexpected 'x': %d", x.Value)
		}
	})

	t.Run("Slice", func(t *testing.T) {
		fn := MustFindFunction(t, prog, "sliceInterface")
		e := NewExecutor(fn)
		defer e.Close()

		// Initial states should run until X1.Val() invocation and then stop on return.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `interface.slice.go:13`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `interface.slice.go:22`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// Initial states should run Y1.Val() invocation and then stop on return.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `interface.slice.go:13`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `interface.slice.go:28`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// Next state should stop at the 'if' block.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `interface.slice.go:13`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// Next state should execute the true 'if' block.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `interface.slice.go:14`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if arrays, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if x, err := EvalVar(state, arrays, values, fn, "x"); err != nil {
			t.Fatal(err)
		} else if y, err := EvalVar(state, arrays, values, fn, "y"); err != nil {
			t.Fatal(err)
		} else if (x.Value + 10) != (y.Value + 20) {
			t.Fatalf("unexpected: 'x'=%d y=%d", x.Value, y.Value)
		}

		// Next state should execute the false 'if' block.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `interface.slice.go:16`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if arrays, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if x, err := EvalVar(state, arrays, values, fn, "x"); err != nil {
			t.Fatal(err)
		} else if y, err := EvalVar(state, arrays, values, fn, "y"); err != nil {
			t.Fatal(err)
		} else if (x.Value + 10) == (y.Value + 20) {
			t.Fatalf("unexpected: 'x'=%d, 'y'=%d", x.Value, y.Value)
		}

		// Ensure available states have been exhausted.
		if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
			t.Fatalf("ExecuteNextState=%s, expected done", err)
		}
	})
}
