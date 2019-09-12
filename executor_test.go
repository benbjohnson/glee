package glee_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"testing"

	"github.com/benbjohnson/glee"
	"github.com/benbjohnson/glee/z3"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// NewExecutor returns a new instance of Executor with a Z3 solver.
func NewExecutor(fn *ssa.Function) *Executor {
	e := &Executor{
		Executor: glee.NewExecutor(fn),
		Solver:   z3.NewSolver(),
	}
	e.Executor.Solver = e.Solver
	return e
}

// Executor is a test wrapper for glee.Executor.
type Executor struct {
	*glee.Executor
	Solver *z3.Solver
}

func (e *Executor) Close() error {
	return e.Solver.Close()
}

// EvalVar is a helper function to evaluate the constant value of a function variable.
func EvalVar(state *glee.ExecutionState, arrays []*glee.Array, values [][]byte, fn *ssa.Function, vname string) (*glee.ConstantExpr, error) {
	binding := state.Eval(MustVarValue(fn, vname))
	if binding == nil {
		return nil, fmt.Errorf("binding not found: %s", vname)
	}
	return glee.NewExprEvaluator(arrays, values).Evaluate(binding.(glee.Expr))
}

// MustBuildProgram builds an SSA program at the given path. Fatal on error.
func MustBuildProgram(tb testing.TB, path string) *ssa.Program {
	tb.Helper()

	// Load the initial set of packages.
	initial, err := packages.Load(&packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: true,
	}, path)
	if err != nil {
		tb.Fatal(err)
	} else if packages.PrintErrors(initial) > 0 {
		tb.Fatal("packages contain errors")
	}

	// Build program in SSA form.
	prog, pkgs := ssautil.AllPackages(initial, ssa.BuilderMode(0))
	for i, pkg := range pkgs {
		if pkg == nil {
			tb.Fatalf("cannot build SSA for package %s", initial[i])
		}
		pkg.SetDebugMode(true)
	}
	prog.Build()

	// Ensure program depends on runtime package.
	if prog.ImportedPackage("runtime") == nil {
		tb.Fatal("program does not depend on runtime")
	}
	return prog
}

// MustFindFunction returns a function from any package in the program with the given name.
func MustFindFunction(tb testing.TB, prog *ssa.Program, name string) *ssa.Function {
	tb.Helper()

	for _, pkg := range prog.AllPackages() {
		if m := pkg.Members[name]; m == nil {
			continue
		} else if fn, ok := m.(*ssa.Function); !ok {
			tb.Fatalf("member %q is %T, not a function", name, m)
		} else {
			return fn
		}
	}
	tb.Fatalf("function %q not found", name)
	return nil
}

// VarValue returns the ssa.Value for a given variable name.
func VarValue(fn *ssa.Function, name string) ssa.Value {
	for _, blk := range fn.Blocks {
		for _, instr := range blk.Instrs {
			if ref, ok := instr.(*ssa.DebugRef); ok {
				if ident, ok := ref.Expr.(*ast.Ident); ok && ident.Name == name {
					return ref.X
				}
			}
		}
	}
	return nil
}

// MustVarValue returns the ssa.Value for a given variable name. Panic if not found.
func MustVarValue(fn *ssa.Function, name string) ssa.Value {
	v := VarValue(fn, name)
	if v == nil {
		panic(fmt.Sprintf("var %q not found", name))
	}
	return v
}

// TrimPosition returns a position with just the base filename and line number.
func TrimPosition(pos token.Position) token.Position {
	if !pos.IsValid() {
		return pos
	}
	pos.Filename = filepath.Base(pos.Filename)
	pos.Column = 0
	return pos
}

func fn2str(fn *ssa.Function) string {
	var buf bytes.Buffer
	fn.WriteTo(&buf)
	return buf.String()
}
