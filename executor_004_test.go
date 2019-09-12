package glee_test

import (
	"testing"

	"github.com/benbjohnson/glee"
)

func TestExecutor_Pkg004_String(t *testing.T) {
	prog := MustBuildProgram(t, "./testdata/pkg004_string")

	t.Run("Concat", func(t *testing.T) {
		fn := MustFindFunction(t, prog, "stringConcat")
		e := NewExecutor(fn)
		defer e.Close()

		// Initial state should run until the 'if' statement.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `concat.go:11`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		}

		// Next state should execute the true 'if' block.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `concat.go:12`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if _, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if got, exp := string(values[0]), "fo"; got != exp {
			t.Fatalf("values[0]=%s, expected %s", got, exp)
		} else if got, exp := string(values[1]), "baz"; got != exp {
			t.Fatalf("values[1]=%s, expected %s", got, exp)
		}

		// Next state should execute the false 'if' block.
		if state, err := e.ExecuteNextState(); err != nil {
			t.Fatal(err)
		} else if got, exp := TrimPosition(state.Position()).String(), `concat.go:14`; got != exp {
			t.Fatalf("unexpected position: %s", got)
		} else if _, values, err := state.Values(); err != nil {
			t.Fatal(err)
		} else if got, exp := string(values[0]), "fo"; got == exp {
			t.Fatalf("values[0]=%s, expected NOT %s", got, exp)
		} else if got, exp := string(values[1]), "baz"; got == exp {
			t.Fatalf("values[1]=%s, expected NOT %s", got, exp)
		}
	})

	t.Run("Convert", func(t *testing.T) {
		t.Run("ByteSlice", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "convertByteSlice")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `convert.byte_slice.go:11`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `convert.byte_slice.go:12`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if got, exp := string(values[0]), "XY"; got != exp {
				t.Fatalf("values[0]=%s, expected %s", got, exp)
			}

			// Next state should execute the false 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `convert.byte_slice.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if got, exp := string(values[0]), "XY"; got == exp {
				t.Fatalf("values[0]=%s, expected NOT %s", got, exp)
			}
		})
	})

	t.Run("Slice", func(t *testing.T) {
		t.Run("OK", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "stringSlice")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `slice.go:10`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `slice.go:11`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if got, exp := string(values[0])[1:3], "XY"; got != exp {
				t.Fatalf("values[0]=%s, expected contains %s", got, exp)
			}

			// Next state should execute the false 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `slice.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if got, exp := string(values[0])[1:3], "XY"; got == exp {
				t.Fatalf("values[0]=%s, expected NOT contains %s", got, exp)
			}
		})

		t.Run("OutOfBounds", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "stringSliceOOB")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `slice.outofbounds.go:10`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if got, exp := state.Status(), glee.ExecutionStatusPanicked; got != exp {
				t.Fatalf("Status()=%s, expected %s", got, exp)
			} else if got, exp := state.Reason(), "slice bounds out of range"; got != exp {
				t.Fatalf("Reason()=%q, expected %q", got, exp)
			}
		})
	})

	t.Run("NEQ", func(t *testing.T) {
		t.Run("ContentMismatch", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "neqContentMismatch")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `neq.content_mismatch.go:11`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `neq.content_mismatch.go:12`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 == value1 {
				t.Fatalf("values: expected %q != %q", value0, value1)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `neq.content_mismatch.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 != value1 {
				t.Fatalf("values: expected %q == %q", value0, value1)
			}
		})
		t.Run("LengthMismatch", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "neqLengthMismatch")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `neq.length_mismatch.go:11`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should ONLY execute the true 'if' block.
			// No values should be returned because it is a constant false.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `neq.length_mismatch.go:12`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if got, exp := len(values), 0; got != exp {
				t.Fatalf("len(values)=%d, expected %d", got, exp)
			}

			// False state should not be accessible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
	})

	t.Run("LSS", func(t *testing.T) {
		t.Run("EqualLen", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "lssEqualLen")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `lss.equal_len.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `lss.equal_len.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 >= value1 {
				t.Fatalf("values: expected %q < %q", value0, value1)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `lss.equal_len.go:16`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 < value1 {
				t.Fatalf("values: expected NOT %q < %q", value0, value1)
			}
		})
		t.Run("Impossible", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "lssImpossible")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `lss.impossible.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `lss.impossible.go:17`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 < value1 {
				t.Fatalf("values: expected NOT %q < %q", value0, value1)
			}

			// No more states as true state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
		t.Run("ShortLHS", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "lssShortLHS")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `lss.short_lhs.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `lss.short_lhs.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 >= value1 {
				t.Fatalf("values: expected %q < %q", value0, value1)
			}

			// No more states as false state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
		t.Run("ShortRHS", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "lssShortRHS")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `lss.short_rhs.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the false 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `lss.short_rhs.go:16`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 < value1 {
				t.Fatalf("values: expected NOT %q < %q", value0, value1)
			}

			// No more states as true state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
	})

	t.Run("LEQ", func(t *testing.T) {
		t.Run("EqualLen", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "leqEqualLen")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `leq.equal_len.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `leq.equal_len.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); !(value0 <= value1) {
				t.Fatalf("values: expected %q <= %q", value0, value1)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `leq.equal_len.go:16`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 <= value1 {
				t.Fatalf("values: expected NOT %q <= %q", value0, value1)
			}
		})
		t.Run("Impossible", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "leqImpossible")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `leq.impossible.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `leq.impossible.go:17`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 <= value1 {
				t.Fatalf("values: expected NOT %q <= %q", value0, value1)
			}

			// No more states as true state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
		t.Run("ShortLHS", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "leqShortLHS")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `leq.short_lhs.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `leq.short_lhs.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 > value1 {
				t.Fatalf("values: expected %q <= %q", value0, value1)
			}

			// No more states as false state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
		t.Run("ShortRHS", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "leqShortRHS")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `leq.short_rhs.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the false 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `leq.short_rhs.go:16`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 <= value1 {
				t.Fatalf("values: expected NOT %q <= %q", value0, value1)
			}

			// No more states as true state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
	})

	t.Run("GTR", func(t *testing.T) {
		t.Run("EqualLen", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "gtrEqualLen")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `gtr.equal_len.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `gtr.equal_len.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); !(value0 > value1) {
				t.Fatalf("values: expected %q > %q", value0, value1)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `gtr.equal_len.go:16`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 > value1 {
				t.Fatalf("values: expected NOT %q > %q", value0, value1)
			}
		})
		t.Run("Impossible", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "gtrImpossible")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `gtr.impossible.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `gtr.impossible.go:17`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 > value1 {
				t.Fatalf("values: expected NOT %q > %q", value0, value1)
			}

			// No more states as true state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
		t.Run("ShortLHS", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "gtrShortLHS")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `gtr.short_lhs.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the false 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `gtr.short_lhs.go:16`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 > value1 {
				t.Fatalf("values: expected NOT %q > %q", value0, value1)
			}

			// No more states as true state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
		t.Run("ShortRHS", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "gtrShortRHS")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `gtr.short_rhs.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `gtr.short_rhs.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); !(value0 > value1) {
				t.Fatalf("values: expected %q > %q", value0, value1)
			}

			// No more states as false state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
	})

	t.Run("GEQ", func(t *testing.T) {
		t.Run("EqualLen", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "geqEqualLen")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `geq.equal_len.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `geq.equal_len.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); !(value0 >= value1) {
				t.Fatalf("values: expected %q <= %q", value0, value1)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `geq.equal_len.go:16`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 >= value1 {
				t.Fatalf("values: expected NOT %q >= %q", value0, value1)
			}
		})
		t.Run("Impossible", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "geqImpossible")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `geq.impossible.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `geq.impossible.go:17`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 >= value1 {
				t.Fatalf("values: expected NOT %q >= %q", value0, value1)
			}

			// No more states as true state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
		t.Run("ShortLHS", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "geqShortLHS")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `geq.short_lhs.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the false 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `geq.short_lhs.go:16`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); value0 >= value1 {
				t.Fatalf("values: expected NOT %q >= %q", value0, value1)
			}

			// No more states as true state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
		t.Run("ShortRHS", func(t *testing.T) {
			fn := MustFindFunction(t, prog, "geqShortRHS")
			e := NewExecutor(fn)
			defer e.Close()

			// Initial state should run until the 'if' statement.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `geq.short_rhs.go:13`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			}

			// Next state should execute the true 'if' block.
			if state, err := e.ExecuteNextState(); err != nil {
				t.Fatal(err)
			} else if got, exp := TrimPosition(state.Position()).String(), `geq.short_rhs.go:14`; got != exp {
				t.Fatalf("unexpected position: %s", got)
			} else if _, values, err := state.Values(); err != nil {
				t.Fatal(err)
			} else if value0, value1 := string(values[0]), string(values[1]); !(value0 >= value1) {
				t.Fatalf("values: expected %q >= %q", value0, value1)
			}

			// No more states as false state is not possible.
			if _, err := e.ExecuteNextState(); err != glee.ErrNoStateAvailable {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
	})
}
