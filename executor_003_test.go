package glee_test

import (
	"testing"
)

func TestExecutor_Pkg003_Slice(t *testing.T) {
	prog := MustBuildProgram(t, "./testdata/pkg003_slice")

	t.Run("ByteSlice", func(t *testing.T) {
		t.Run("Slice", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "sliceByteSlice")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `byte_slice.go:12`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `byte_slice.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if got, exp := string(values[0][1:3]), "XY"; got != exp {
				t.Fatalf("values[0]=%s, expected %s", got, exp)
			}

			// Next state should execute the false 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `byte_slice.go:15`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if got, exp := string(values[0][1:3]), "XY"; got == exp {
				t.Fatalf("values[0]=%s, expected NOT %s", got, exp)
			}
		})

		t.Run("IndexAddr", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "byteSliceIndexAddr")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `byte_slice.index_addr.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `byte_slice.index_addr.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if got, exp := string(values[0][1:3]), "YX"; got != exp {
				t.Fatalf("values[0]=%s, expected %s", got, exp)
			}

			// Next state should execute the false 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `byte_slice.index_addr.go:16`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if got, exp := string(values[0][1:3]), "YX"; got == exp {
				t.Fatalf("values[0]=%s, expected NOT %s", got, exp)
			}
		})

		t.Run("Make", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "byteSliceMake")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `byte_slice.make.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `byte_slice.make.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if got, exp := string(values[0]), "X"; got != exp {
				t.Fatalf("values[0]=%s, expected %s", got, exp)
			} else if got, exp := string(values[1]), "Y"; got != exp {
				t.Fatalf("values[1]=%s, expected %s", got, exp)
			}

			// Next state should execute the false 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `byte_slice.make.go:16`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if got, exp := string(values[0])+string(values[1]), "XY"; got == exp {
				t.Fatalf("values[0..1]=%s, expected NOT %s", got, exp)
			}
		})
	})
}
