package glee

import (
	"errors"
	"fmt"
)

// Standard widths.
const (
	WidthBool = 1
	Width8    = 8
	Width16   = 16
	Width32   = 32
	Width64   = 64
)

var (
	ErrSolverTimeout       = errors.New("Solver timeout")
	ErrSolverCanceled      = errors.New("Solver canceled")
	ErrSolverResourceLimit = errors.New("Solver resource limit")
	ErrSolverUnknown       = errors.New("Solver unknown error")
)

// assert panics if condition is false.
func assert(condition bool, format string, args ...interface{}) {
	if !condition {
		panic(fmt.Sprintf("assert: "+format, args...))
	}
}
