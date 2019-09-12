package glee_test

import (
	"testing"
)

func TestExecutor_Pkg001_Call(t *testing.T) {
	prog := MustBuildProgram(t, "./testdata/pkg001_call")

	t.Run("Simple", func(t *testing.T) {
		caller := MustFindFunction(t, prog, "caller")
		callee := MustFindFunction(t, prog, "callee")
		e := NewExecutor(caller)
		defer e.Close()

		// Initial state should stop at call to callee().
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `simple.go:10`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// Next state should stop at 'if' in callee().
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `simple.go:18`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// Next state should run from callee() true to end of callee().
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `simple.go:19`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if arrays, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if x, err := EvalVar(state, arrays, values, callee, "x"); err != nil {
			t.Fatal(err)
		} else if x.Value <= 10 {
			t.Fatalf("unexpected 'x': %d", x.Value)
		}

		// Next state should run until caller() 'if'
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `simple.go:11`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// Next state should execute caller() 'if': true condition.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `simple.go:12`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if arrays, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if x, err := EvalVar(state, arrays, values, caller, "x"); err != nil {
			t.Fatal(err)
		} else if y, err := EvalVar(state, arrays, values, caller, "y"); err != nil {
			t.Fatal(err)
		} else if x8, y16 := int8(x.Value), int16(y.Value); int32(x8)*int32(y16)+1 != 0xAABB {
			t.Fatalf("unexpected 'x' & 'y': %d, %d", x8, y16)
		}

		// Next state should execute caller() false. Implicit return has no position.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `-`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if arrays, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if x, err := EvalVar(state, arrays, values, caller, "x"); err != nil {
			t.Fatal(err)
		} else if y, err := EvalVar(state, arrays, values, caller, "y"); err != nil {
			t.Fatal(err)
		} else if x8, y16 := int8(x.Value), int16(y.Value); int32(x8)*int32(y16)+1 == 0xAABB {
			t.Fatalf("unexpected 'x' & 'y': %d, %d", x8, y16)
		}

		// Next state should execute callee() false until end of callee()
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `simple.go:21`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if arrays, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if x, err := EvalVar(state, arrays, values, callee, "x"); err != nil {
			t.Fatal(err)
		} else if x32 := int32(x.Value); x32 > 10 {
			t.Fatalf("unexpected 'x': %d", x32)
		}

		// Next state should run until caller() 'if'
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `simple.go:11`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// Next state should execute caller() false. The true condition is impossible.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `-`; got != exp { // implicit return
			t.Fatalf("unexpected position: %s", got)
		} else if arrays, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if x, err := EvalVar(state, arrays, values, caller, "x"); err != nil {
			t.Fatal(err)
		} else if y, err := EvalVar(state, arrays, values, caller, "y"); err != nil {
			t.Fatal(err)
		} else if x8, y16 := int8(x.Value), int16(y.Value); int32(x8)*int32(y16)+1 == 0xAABB {
			t.Fatalf("unexpected 'x' & 'y': %d, %d", x8, y16)
		} else if x8, y16 := int8(x.Value), int16(y.Value); int32(x8)*int32(y16)+1 == 0xAABB {
			t.Fatalf("unexpected 'x' & 'y': %d, %d", x8, y16)
		}
	})
}
