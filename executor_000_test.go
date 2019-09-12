package glee_test

import (
	"encoding/hex"
	"testing"

	"github.com/benbjohnson/glee"
)

func TestExecutor_Pkg000_If(t *testing.T) {
	prog := MustBuildProgram(t, "./testdata/pkg000_if")

	t.Run("Simple", func(t *testing.T) {
		fn := MustFindFunction(t, prog, "simple")
		e := NewExecutor(fn)
		defer e.Close()

		// Initial state should create a symbolic 'x' value and stop at the 'if'.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if binding := state.Eval(MustVarValue(fn, "x")); binding == nil {
			t.Fatal("binding for 'x' not found")
		}

		// Next state hold the true condition ('x == 100').
		state, err := e.ExecuteNextState()
		if err != nil {
			t.Fatal(err)
		} else if got, exp := len(state.Constraints()), 1; got != exp {
			t.Fatalf("len(ExecutionState.Constraints())=%d, expected %d", got, exp)
		} else if constraint, ok := state.Constraints()[0].(*glee.BinaryExpr); !ok || constraint.Op != glee.EQ {
			t.Fatalf("expected EQ constraint, got %s", state.Constraints()[0])
		}

		// Solve for 'x'.
		if arrays, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if got, exp := len(arrays), 1; got != exp {
			t.Fatalf("len(arrays)=%d, expected %d", got, exp)
		} else if got, exp := hex.EncodeToString(values[0]), "bbaa000000000000"; got != exp { // 64-bit litte-endian
			t.Fatalf("values[0]=%s, expected %s", got, exp)
		}

		// Next state hold the false condition ('x != 100').
		state, err = e.ExecuteNextState()
		if err != nil {
			t.Fatal(err)
		} else if got, exp := len(state.Constraints()), 1; got != exp {
			t.Fatalf("len(ExecutionState.Constraints())=%d, expected %d", got, exp)
		} else if _, ok := state.Constraints()[0].(*glee.NotExpr); !ok {
			t.Fatalf("expected NOT constraint, got %s", state.Constraints()[0])
		}

		// Solve for 'x'.
		if arrays, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if got, exp := len(arrays), 1; got != exp {
			t.Fatalf("len(arrays)=%d, expected %d", got, exp)
		} else if got := hex.EncodeToString(values[0]); got == `bbaa000000000000` { // 64-bit litte-endian
			t.Fatalf("values[0]=%s, expected any other value", got)
		}
	})
}
