package z3

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/benbjohnson/glee"
)

/*
#cgo LDFLAGS: -lz3
#include <z3.h>
#include <stdlib.h>
#include <stdio.h>
*/
import "C"

// Ensure solver implements interface.
var _ glee.Solver = (*Solver)(nil)

// Solver represents a solver that uses an embedded Z3 solver.
type Solver struct {
	ctx   *Context
	stats Stats
}

// NewSolver returns a new instance of Solver.
func NewSolver() *Solver {
	return &Solver{
		ctx: NewContext(),
	}
}

// Close deletes the underlying Z3 context.
func (s *Solver) Close() error {
	return s.ctx.Close()
}

// Stats returns statistics for the solver.
func (s *Solver) Stats() Stats {
	return s.stats
}

func (s *Solver) Solve(constraints []glee.Expr, arrays []*glee.Array) (satisfiable bool, values [][]byte, err error) {
	t := time.Now()
	defer func() {
		s.stats.SolveN++
		s.stats.SolveTime += time.Since(t)
	}()

	solver := C.Z3_mk_solver(s.ctx.raw)
	if err := s.ctx.err("Z3_mk_solver"); err != nil {
		return false, nil, err
	}
	C.Z3_solver_inc_ref(s.ctx.raw, solver)
	defer C.Z3_solver_dec_ref(s.ctx.raw, solver)

	// Assert constraints.
	// println("dbg/solve", len(constraints))
	for _, constraint := range constraints {
		z3Constraint, err := s.ctx.toAST(constraint)
		if err != nil {
			return false, nil, err
		}
		C.Z3_solver_assert(s.ctx.raw, solver, z3Constraint)
		if err := s.ctx.err("Z3_solver_assert"); err != nil {
			return false, nil, err
		}
		// println("dbg/solve.assert\n", s.ctx.astToString(z3Constraint))
	}

	// Check equations with the solver.
	// Exit immediately if unsatisfiable or the solver encountered an error.
	ret := C.Z3_solver_check(s.ctx.raw, solver)
	if err := s.ctx.err("Z3_solver_check"); err != nil {
		return false, nil, err
	} else if ret == C.Z3_L_FALSE {
		return false, nil, nil
	} else if ret == C.Z3_L_UNDEF {
		reason := C.GoString(C.Z3_solver_get_reason_unknown(s.ctx.raw, solver))
		switch {
		case strings.Contains(reason, "timeout"):
			return false, nil, glee.ErrSolverTimeout
		case strings.Contains(reason, "canceled"):
			return false, nil, glee.ErrSolverCanceled
		case strings.Contains(reason, "(resource limits reached)"):
			return false, nil, glee.ErrSolverResourceLimit
		case strings.Contains(reason, "unknown"):
			return false, nil, glee.ErrSolverUnknown
		default:
			return false, nil, fmt.Errorf("z3: %s", reason)
		}
	} else if len(arrays) == 0 {
		return true, nil, nil // no symbolics, ignore model
	}

	// Calculate a model for the given formula.
	model := C.Z3_solver_get_model(s.ctx.raw, solver)
	if err := s.ctx.err("Z3_solver_get_model"); err != nil {
		return true, nil, err
	}
	// println("dbg/model\n", s.ctx.modelToString(model))

	// Fetch values for symbolic arrays.
	values, err = s.ctx.eval(model, arrays)
	if err != nil {
		return true, nil, err
	}
	return true, values, nil
}

// Context represents a Z3 context object that is used for constructing expressions.
type Context struct {
	raw C.Z3_context
}

// NewContext returns a new instance of Context.
func NewContext() *Context {
	config := C.Z3_mk_config()
	defer C.Z3_del_config(config)

	raw := C.Z3_mk_context(config)
	C.Z3_set_error_handler(raw, nil)
	C.Z3_set_ast_print_mode(raw, C.Z3_PRINT_SMTLIB2_COMPLIANT)
	return &Context{raw: raw}
}

// Close deletes the underlying Z3 context.
func (ctx *Context) Close() error {
	C.Z3_del_context(ctx.raw)
	return ctx.err("Z3_del_context")
}

// err returns the error for the last API call. Returns nil if last call was successful.
func (ctx *Context) err(op string) error {
	if code := C.Z3_get_error_code(ctx.raw); code != C.Z3_OK {
		return &Error{Code: int(code), Op: op, Message: C.GoString(C.Z3_get_error_msg(ctx.raw, code))}
	}
	return nil
}

// toAST returns a new instance of Z3_ast and its width from a glee expression.
func (ctx *Context) toAST(expr glee.Expr) (C.Z3_ast, error) {
	switch expr := expr.(type) {
	case *glee.ConstantExpr:
		return ctx.toConstantAST(expr)
	case *glee.NotOptimizedExpr:
		return ctx.toAST(expr.Src)
	case *glee.SelectExpr:
		return ctx.toSelectAST(expr)
	case *glee.ConcatExpr:
		return ctx.toConcatAST(expr)
	case *glee.ExtractExpr:
		return ctx.toExtractAST(expr)
	case *glee.CastExpr:
		return ctx.toCastAST(expr)
	case *glee.NotExpr:
		return ctx.toNotAST(expr)
	case *glee.BinaryExpr:
		return ctx.toBinaryAST(expr)
	default:
		return nil, fmt.Errorf("ctx.Context.toAST: invalid expression type: %T", expr)
	}
}

func (ctx *Context) toConstantAST(expr *glee.ConstantExpr) (C.Z3_ast, error) {
	if expr.Width == 1 {
		if expr.IsTrue() {
			return ctx.makeTrue()
		}
		return ctx.makeFalse()
	} else if expr.Width <= 32 {
		return ctx.makeUint(expr.Width, uint32(expr.Value))
	} else if expr.Width <= 64 {
		return ctx.makeUint64(expr.Width, expr.Value)
	}
	return nil, fmt.Errorf("z3.Context.toConstantAST: invalid expression width: %d", expr.Width)
}

func (ctx *Context) toSelectAST(expr *glee.SelectExpr) (C.Z3_ast, error) {
	array, err := ctx.makeArrayWithUpdate(expr.Array, expr.Array.Updates)
	if err != nil {
		return nil, err
	}
	index, err := ctx.toAST(expr.Index)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_select(ctx.raw, array, index), ctx.err("Z3_mk_select")
}

func (ctx *Context) toConcatAST(expr *glee.ConcatExpr) (C.Z3_ast, error) {
	msb, err := ctx.toAST(expr.MSB)
	if err != nil {
		return nil, err
	}
	lsb, err := ctx.toAST(expr.LSB)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_concat(ctx.raw, msb, lsb), ctx.err("Z3_mk_concat")
}

func (ctx *Context) toExtractAST(expr *glee.ExtractExpr) (C.Z3_ast, error) {
	src, err := ctx.toAST(expr.Expr)
	if err != nil {
		return nil, err
	}

	// If extracting single bit, use EQ expression to convert to bool sort.
	if expr.Width == 1 {
		extractExpr := C.Z3_mk_extract(ctx.raw, C.uint(expr.Offset), C.uint(expr.Offset), src)
		if err := ctx.err("Z3_mk_extract[bool]"); err != nil {
			return nil, err
		}
		one, err := ctx.makeUint64(1, 1)
		if err != nil {
			return nil, err
		}
		return C.Z3_mk_eq(ctx.raw, extractExpr, one), ctx.err("Z3_mk_eq")
	}

	//
	return C.Z3_mk_extract(ctx.raw, C.uint(expr.Offset+expr.Width-1), C.uint(expr.Offset), src), ctx.err("Z3_mk_extract")
}

func (ctx *Context) toCastAST(expr *glee.CastExpr) (C.Z3_ast, error) {
	if expr.Signed {
		return ctx.toSignedCastAST(expr)
	}
	return ctx.toUnsignedCastAST(expr)
}

func (ctx *Context) toSignedCastAST(expr *glee.CastExpr) (C.Z3_ast, error) {
	src, err := ctx.toAST(expr.Src)
	if err != nil {
		return nil, err
	}

	// Convert boolean cast to if-then-else expression.
	if glee.ExprWidth(expr.Src) == 1 {
		minusOne := int64(-1)
		whenTrue, err := ctx.makeUint64(expr.Width, uint64(minusOne))
		if err != nil {
			return nil, err
		}
		whenFalse, err := ctx.makeUint64(expr.Width, 0)
		if err != nil {
			return nil, err
		}
		return C.Z3_mk_ite(ctx.raw, src, whenTrue, whenFalse), ctx.err("Z3_mk_ite")
	}

	// Otherwise return sign-extension.
	return C.Z3_mk_sign_ext(ctx.raw, C.uint(expr.Width-uint(ctx.bvSize(src))), src), ctx.err("Z3_mk_sign_ext")
}

func (ctx *Context) toUnsignedCastAST(expr *glee.CastExpr) (C.Z3_ast, error) {
	src, err := ctx.toAST(expr.Src)
	if err != nil {
		return nil, err
	}

	// Convert boolean cast to if-then-else expression.
	if glee.ExprWidth(expr.Src) == 1 {
		whenTrue, err := ctx.makeUint64(expr.Width, 1)
		if err != nil {
			return nil, err
		}
		whenFalse, err := ctx.makeUint64(expr.Width, 0)
		if err != nil {
			return nil, err
		}
		return C.Z3_mk_ite(ctx.raw, src, whenTrue, whenFalse), ctx.err("Z3_mk_ite")
	}

	// Otherwise return zero-padding bit vector.
	padding, err := ctx.makeUint64(expr.Width-ctx.bvSize(src), 0)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_concat(ctx.raw, padding, src), ctx.err("Z3_mk_concat")
}

func (ctx *Context) toNotAST(expr *glee.NotExpr) (C.Z3_ast, error) {
	src, err := ctx.toAST(expr.Expr)
	if err != nil {
		return nil, err
	}

	// If boolean, use boolean NOT operation.
	if glee.ExprWidth(expr.Expr) == 1 {
		return C.Z3_mk_not(ctx.raw, src), ctx.err("Z3_mk_not")
	}
	return C.Z3_mk_bvnot(ctx.raw, src), ctx.err("Z3_mk_bvnot")
}

func (ctx *Context) toBinaryAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	switch expr.Op {
	case glee.ADD:
		return ctx.toBinaryAddAST(expr)
	case glee.SUB:
		return ctx.toBinarySubAST(expr)
	case glee.MUL:
		return ctx.toBinaryMulAST(expr)
	case glee.UDIV:
		return ctx.toBinaryUDivAST(expr)
	case glee.SDIV:
		return ctx.toBinarySDivAST(expr)
	case glee.UREM:
		return ctx.toBinaryURemAST(expr)
	case glee.SREM:
		return ctx.toBinarySRemAST(expr)
	case glee.AND:
		return ctx.toBinaryAndAST(expr)
	case glee.OR:
		return ctx.toBinaryOrAST(expr)
	case glee.XOR:
		return ctx.toBinaryXorAST(expr)
	case glee.SHL:
		return ctx.toBinaryShlAST(expr)
	case glee.LSHR:
		return ctx.toBinaryLShrAST(expr)
	case glee.ASHR:
		return ctx.toBinaryAShrAST(expr)
	case glee.EQ:
		return ctx.toBinaryEqAST(expr)
	case glee.ULT:
		return ctx.toBinaryUltAST(expr)
	case glee.ULE:
		return ctx.toBinaryUleAST(expr)
	case glee.SLT:
		return ctx.toBinarySltAST(expr)
	case glee.SLE:
		return ctx.toBinarySleAST(expr)
	default:
		return nil, fmt.Errorf("ctx.Context.toBinaryExpr: unexpected operation: %s", expr.Op)
	}
}

func (ctx *Context) toBinaryAddAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvadd(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvadd")
}

func (ctx *Context) toBinarySubAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvsub(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvsub")
}

func (ctx *Context) toBinaryMulAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvmul(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvmul")
}

func (ctx *Context) toBinaryUDivAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvudiv(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvudiv")
}

func (ctx *Context) toBinarySDivAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvsdiv(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvsdiv")
}

func (ctx *Context) toBinaryURemAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvurem(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvurem")
}

func (ctx *Context) toBinarySRemAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvsrem(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvsrem")
}

func (ctx *Context) toBinaryAndAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}

	if glee.ExprWidth(expr.LHS) == 1 {
		args := [2]C.Z3_ast{lhs, rhs}
		return C.Z3_mk_and(ctx.raw, 2, &args[0]), ctx.err("Z3_mk_and")
	}
	return C.Z3_mk_bvand(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvand")
}

func (ctx *Context) toBinaryOrAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}

	if glee.ExprWidth(expr.LHS) == 1 {
		args := [2]C.Z3_ast{lhs, rhs}
		return C.Z3_mk_or(ctx.raw, 2, &args[0]), ctx.err("Z3_mk_or")
	}
	return C.Z3_mk_bvor(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvor")
}

func (ctx *Context) toBinaryXorAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}

	if glee.ExprWidth(expr.LHS) == 1 {
		notRHS, err := C.Z3_mk_not(ctx.raw, rhs)
		if err != nil {
			return nil, err
		}
		return C.Z3_mk_ite(ctx.raw, lhs, notRHS, rhs), ctx.err("Z3_mk_ite")
	}

	return C.Z3_mk_bvxor(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvxor")
}

func (ctx *Context) toBinaryShlAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvshl(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvshl")
}

func (ctx *Context) toBinaryLShrAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvlshr(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvlshr")
}

func (ctx *Context) toBinaryAShrAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvashr(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvashr")
}

func (ctx *Context) toBinaryEqAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	if glee.ExprWidth(expr.LHS) == 1 {
		return C.Z3_mk_iff(ctx.raw, lhs, rhs), ctx.err("Z3_mk_iff")
	}
	return C.Z3_mk_eq(ctx.raw, lhs, rhs), ctx.err("Z3_mk_eq")
}

func (ctx *Context) toBinaryUltAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvult(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvult")
}

func (ctx *Context) toBinaryUleAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvule(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvule")
}

func (ctx *Context) toBinarySltAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvslt(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvslt")
}

func (ctx *Context) toBinarySleAST(expr *glee.BinaryExpr) (C.Z3_ast, error) {
	lhs, err := ctx.toAST(expr.LHS)
	if err != nil {
		return nil, err
	}
	rhs, err := ctx.toAST(expr.RHS)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_bvsle(ctx.raw, lhs, rhs), ctx.err("Z3_mk_bvsle")
}

func (ctx *Context) makeTrue() (C.Z3_ast, error) {
	return C.Z3_mk_true(ctx.raw), ctx.err("Z3_mk_true")
}

func (ctx *Context) makeFalse() (C.Z3_ast, error) {
	return C.Z3_mk_false(ctx.raw), ctx.err("Z3_mk_false")
}

func (ctx *Context) makeBVSort(width uint) (C.Z3_sort, error) {
	return C.Z3_mk_bv_sort(ctx.raw, C.uint(width)), ctx.err("Z3_mk_bv_sort")
}

func (ctx *Context) makeUint(width uint, value uint32) (C.Z3_ast, error) {
	t, err := ctx.makeBVSort(width)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_unsigned_int(ctx.raw, C.uint(value), t), ctx.err("Z3_mk_unsigned_int")
}

func (ctx *Context) makeUint64(width uint, value uint64) (C.Z3_ast, error) {
	t, err := ctx.makeBVSort(width)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_unsigned_int64(ctx.raw, C.ulonglong(value), t), ctx.err("Z3_mk_unsigned_int64")
}

func (ctx *Context) bvSize(expr C.Z3_ast) uint {
	t := C.Z3_get_sort(ctx.raw, expr)
	if err := ctx.err("Z3_get_sort"); err != nil {
		panic(err)
	}
	return ctx.bvSortSize(t)
}

// bvSortSize returns the size of t in bits. Panic if t is not a bit-vector sort.
func (ctx *Context) bvSortSize(t C.Z3_sort) uint {
	sz := uint(C.Z3_get_bv_sort_size(ctx.raw, t))
	if err := ctx.err("Z3_get_bv_sort_size"); err != nil {
		panic(err)
	}
	return sz
}

// makeArrayConst returns the root constant array with no updates.
func (ctx *Context) makeArrayConst(array *glee.Array) (C.Z3_ast, error) {
	// Construct array sort.
	domainSort := C.Z3_mk_bv_sort(ctx.raw, C.uint(glee.Width64))
	if err := ctx.err("Z3_mk_bv_sort[domain]"); err != nil {
		return nil, err
	}
	rangeSort := C.Z3_mk_bv_sort(ctx.raw, C.uint(glee.Width8))
	if err := ctx.err("Z3_mk_bv_sort[range]"); err != nil {
		return nil, err
	}
	arraySort := C.Z3_mk_array_sort(ctx.raw, domainSort, rangeSort)
	if err := ctx.err("Z3_mk_array_sort"); err != nil {
		return nil, err
	}

	// Construct Z3 string for name.
	cname := C.CString(arrayName(array))
	defer C.free(unsafe.Pointer(cname))
	nameSymbol := C.Z3_mk_string_symbol(ctx.raw, cname)

	return C.Z3_mk_const(ctx.raw, nameSymbol, arraySort), ctx.err("Z3_mk_const")
}

// makeArrayWithUpdate returns an array with updates recursively applied.
func (ctx *Context) makeArrayWithUpdate(root *glee.Array, upd *glee.ArrayUpdate) (C.Z3_ast, error) {
	if upd == nil {
		return ctx.makeArrayConst(root)
	}

	array, err := ctx.makeArrayWithUpdate(root, upd.Next)
	if err != nil {
		return nil, err
	}
	index, err := ctx.toAST(upd.Index)
	if err != nil {
		return nil, err
	}
	value, err := ctx.toAST(upd.Value)
	if err != nil {
		return nil, err
	}
	return C.Z3_mk_store(ctx.raw, array, index, value), ctx.err("Z3_mk_store")
}

// eval evaluates arrays into their initial byte slice values.
func (ctx *Context) eval(model C.Z3_model, arrays []*glee.Array) ([][]byte, error) {
	values := make([][]byte, 0, len(arrays))
	for _, array := range arrays {
		value, err := ctx.evalArray(model, array)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// evalArray evaluates a single array into its initial byte slice value.
func (ctx *Context) evalArray(model C.Z3_model, array *glee.Array) ([]byte, error) {
	value := make([]byte, 0, array.Size)
	for offset := uint(0); offset < array.Size; offset++ {
		// Generate a reference to the root array.
		z3Array, err := ctx.makeArrayConst(array)
		if err != nil {
			return nil, err
		}
		z3Offset, err := ctx.makeUint64(64, uint64(offset))
		if err != nil {
			return nil, err
		}

		// Generate an expression to select a single byte from the array.
		z3Select := C.Z3_mk_select(ctx.raw, z3Array, z3Offset)
		if err := ctx.err("Z3_mk_select"); err != nil {
			return nil, err
		}

		// Evaluate the expression against the Z3 model.
		var z3Expr C.Z3_ast
		C.Z3_model_eval(ctx.raw, model, z3Select, C.bool(true), &z3Expr)
		if err := ctx.err("Z3_model_eval"); err != nil {
			return nil, err
		}

		// Extract the byte from the evaluation.
		var z3Byte C.int
		C.Z3_get_numeral_int(ctx.raw, z3Expr, &z3Byte)
		if err := ctx.err("Z3_get_numeral_int"); err != nil {
			return nil, err
		}
		value = append(value, byte(z3Byte))
	}
	return value, nil
}

func (ctx *Context) astToString(ast C.Z3_ast) string {
	return C.GoString(C.Z3_ast_to_string(ctx.raw, ast))
}

func (ctx *Context) astSortToString(ast C.Z3_ast) string {
	return ctx.sortToString(C.Z3_get_sort(ctx.raw, ast))
}

func (ctx *Context) sortToString(t C.Z3_sort) string {
	return C.GoString(C.Z3_sort_to_string(ctx.raw, t))
}

func (ctx *Context) modelToString(model C.Z3_model) string {
	return C.GoString(C.Z3_model_to_string(ctx.raw, model))
}

func arrayName(array *glee.Array) string {
	return fmt.Sprintf("A%d", array.ID)
}

func assert(condition bool) {
	if !condition {
		panic("assert failed")
	}
}

// Error represents an error from the Z3 API.
type Error struct {
	Code    int
	Op      string
	Message string
}

// Error returns the error as a string.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s (%d)", e.Op, e.Message, e.Code)
}

// Possible error codes.
const (
	ErrorCodeOK = iota
	ErrorCodeSortError
	ErrorCodeIOB
	ErrorCodeInvalidArg
	ErrorCodeParserError
	ErrorCodeNoParser
	ErrorCodeInvalidPattern
	ErrorCodeMemoutFail
	ErrorCodeFileAccessError
	ErrorCodeInternalFatal
	ErrorCodeInvalidUsage
	ErrorCodeDecRefError
	ErrorCodeException
)

type Stats struct {
	SolveN    int
	SolveTime time.Duration
}
