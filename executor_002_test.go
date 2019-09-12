package glee_test

import (
	"encoding/hex"
	"testing"
)

func TestExecutor_Pkg002_Struct(t *testing.T) {
	prog := MustBuildProgram(t, "./testdata/pkg002_struct")

	t.Run("Simple", func(t *testing.T) {
		fn := MustFindFunction(t, prog, "simple")
		e := NewExecutor(fn)
		defer e.Close()

		// Initial state should run until the 'if' statement.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `simple.go:14`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// Next state should execute the true 'if' block.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `simple.go:15`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if _, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if got, exp := hex.EncodeToString(values[0]), "0200000000000000"; got != exp { // 64-bit litte-endian
			t.Fatalf("values[0]=%s, expected %s", got, exp)
		}

		// Next state should execute the false 'if' block.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `simple.go:17`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if _, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if got, exp := hex.EncodeToString(values[0]), "0200000000000000"; got == exp { // 64-bit litte-endian
			t.Fatalf("values[0]=%s, expected NOT %s", got, exp)
		}
	})
}
