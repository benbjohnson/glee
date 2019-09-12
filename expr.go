package glee

import (
	"bytes"
	"fmt"
	"sort"
)

// Expr represents a symbolic expression.
type Expr interface {
	Binding
	expr()
}

func (*BinaryExpr) expr()       {}
func (*CastExpr) expr()         {}
func (*ConcatExpr) expr()       {}
func (*ConstantExpr) expr()     {}
func (*ExtractExpr) expr()      {}
func (*NotExpr) expr()          {}
func (*NotOptimizedExpr) expr() {}
func (*SelectExpr) expr()       {}

// ExprWidth returns the bit width of the expression.
func ExprWidth(expr Expr) uint {
	switch expr := expr.(type) {
	case *ConstantExpr:
		return expr.Width
	case *NotOptimizedExpr:
		return ExprWidth(expr.Src)
	case *SelectExpr:
		return Width8
	case *ConcatExpr:
		return ExprWidth(expr.MSB) + ExprWidth(expr.LSB)
	case *ExtractExpr:
		return expr.Width
	case *NotExpr:
		return ExprWidth(expr.Expr)
	case *CastExpr:
		return expr.Width
	case *BinaryExpr:
		if expr.Op.IsCompare() {
			return WidthBool
		}
		return ExprWidth(expr.LHS)
	default:
		panic("unreachable")
	}
}

// BinaryOp represents a binary expression operations.
type BinaryOp int

// BinaryExpr operations.
const (
	arithmetic_op_begin = BinaryOp(iota)
	ADD
	SUB
	MUL
	UDIV
	SDIV
	UREM
	SREM
	AND
	OR
	XOR
	SHL
	LSHR
	ASHR
	arithmetic_op_end

	compare_op_begin
	EQ
	NE
	ULT
	ULE
	UGT
	UGE
	SLT
	SLE
	SGT
	SGE
	compare_op_end
)

var binaryOps = [...]string{
	ADD:  "add",
	SUB:  "sub",
	MUL:  "mul",
	UDIV: "udiv",
	SDIV: "sdiv",
	UREM: "urem",
	SREM: "srem",
	AND:  "and",
	OR:   "or",
	XOR:  "xor",
	SHL:  "shl",
	LSHR: "lshr",
	ASHR: "ashr",
	EQ:   "eq",
	NE:   "ne",
	ULT:  "ult",
	ULE:  "ule",
	UGT:  "ugt",
	UGE:  "uge",
	SLT:  "slt",
	SLE:  "sle",
	SGT:  "sgt",
	SGE:  "sge",
}

// String returns the string representation of the operation.
func (op BinaryOp) String() string {
	if op >= 0 && op < BinaryOp(len(binaryOps)) && binaryOps[op] != "" {
		return binaryOps[op]
	}
	return fmt.Sprintf("BinaryOp<%d>", op)
}

// IsArithmetic returns true if op is an arithmetic operator.
func (op BinaryOp) IsArithmetic() bool {
	return op > arithmetic_op_begin && op < arithmetic_op_end
}

// IsCompare returns true if op is a comparison operator.
func (op BinaryOp) IsCompare() bool {
	return op > compare_op_begin && op < compare_op_end
}

// BinaryExpr represents an operation on two expressions.
type BinaryExpr struct {
	Op  BinaryOp
	LHS Expr
	RHS Expr
}

// BinaryExpr returns a new instance of BinaryExpr.
func NewBinaryExpr(op BinaryOp, lhs, rhs Expr) Expr {
	// assert(ExprWidth(lhs) == ExprWidth(rhs), "binary expr width mismatch: op=%s (%T) %d != (%T) %d", op, lhs, ExprWidth(lhs), rhs, ExprWidth(rhs))

	switch op {
	// Arithmetic operators
	case ADD:
		return newAddExpr(lhs, rhs)
	case SUB:
		return newSubExpr(lhs, rhs)
	case MUL:
		return newMulExpr(lhs, rhs)
	case UDIV, SDIV:
		return newDivExpr(op, lhs, rhs)
	case UREM, SREM:
		return newRemExpr(op, lhs, rhs)
	case AND:
		return newAndExpr(lhs, rhs)
	case OR:
		return newOrExpr(lhs, rhs)
	case XOR:
		return newXorExpr(lhs, rhs)
	case SHL:
		return newShlExpr(lhs, rhs)
	case LSHR:
		return newLShrExpr(lhs, rhs)
	case ASHR:
		return newAShrExpr(lhs, rhs)

	// Comparison operators
	case EQ:
		return newEqExpr(lhs, rhs)
	case NE:
		return NewBinaryExpr(EQ, NewConstantExpr(0, WidthBool), NewBinaryExpr(EQ, lhs, rhs))
	case ULT:
		return newUltExpr(lhs, rhs)
	case UGT:
		return newUltExpr(rhs, lhs) // reverse
	case ULE:
		return newUleExpr(lhs, rhs)
	case UGE:
		return newUleExpr(rhs, lhs) // reverse
	case SLT:
		return newSltExpr(lhs, rhs)
	case SGT:
		return newSltExpr(rhs, lhs) // reverse
	case SLE:
		return newSleExpr(lhs, rhs)
	case SGE:
		return newSleExpr(rhs, lhs) // reverse

	default:
		panic("unreachable")
	}
}

// String returns the string representation of the expression.
func (e *BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", e.Op, e.LHS, e.RHS)
}

// newAddExpr returns the expression representing the sum of lhs & rhs.
func newAddExpr(lhs, rhs Expr) Expr {
	// Move constant expression to left hand side.
	if !IsConstantExpr(lhs) && IsConstantExpr(rhs) {
		lhs, rhs = rhs, lhs
	}

	// Refactor to XOR for boolean expressions.
	if ExprWidth(lhs) == WidthBool {
		return NewBinaryExpr(XOR, lhs, rhs)
	}

	// Compute constant if both sides are constant.
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if lhs.Value == 0 {
			return rhs
		} else if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.Add(rhs)
		}
	}

	// Merge constant LHS with constant in RHS binary expression.
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*BinaryExpr); ok {
			if rhs.Op == ADD && IsConstantExpr(rhs.LHS) { // X + (Y+z) == (X+Y) + z
				return NewBinaryExpr(ADD, NewBinaryExpr(ADD, lhs, rhs.LHS), rhs.RHS)
			} else if rhs.Op == SUB && IsConstantExpr(rhs.LHS) { // X + (Y-z) == (X+Y) - z
				return NewBinaryExpr(SUB, NewBinaryExpr(ADD, lhs, rhs.LHS), rhs.RHS)
			}
		}
	}

	// Refactor constant LHS.LHS to a standalone value on LHS.
	if lhs, ok := lhs.(*BinaryExpr); ok && IsConstantExpr(lhs.LHS) {
		if lhs.Op == ADD { // (X+y) + z = X + (y+z)
			return NewBinaryExpr(ADD, lhs.LHS, NewBinaryExpr(ADD, lhs.RHS, rhs))
		} else if lhs.Op == SUB { // (x-y) + z = x + (z-y)
			return NewBinaryExpr(ADD, lhs.LHS, NewBinaryExpr(SUB, rhs, lhs.RHS))
		}
	}

	// Refactor constant RHS.LHS to a standalone value on LHS.
	if rhs, ok := rhs.(*BinaryExpr); ok && IsConstantExpr(rhs.LHS) {
		if rhs.Op == ADD { // a + (k+b) = k+(a+b)
			return NewBinaryExpr(ADD, rhs.LHS, NewBinaryExpr(ADD, lhs, rhs.RHS))
		} else if rhs.Op == SUB { // a + (k-b) = k+(a-b)
			return NewBinaryExpr(ADD, rhs.LHS, NewBinaryExpr(SUB, lhs, rhs.RHS))
		}
	}

	return &BinaryExpr{Op: ADD, LHS: lhs, RHS: rhs}
}

// newSubExpr returns an expression representing the difference of lhs & rhs.
func newSubExpr(lhs, rhs Expr) Expr {
	// Subtracting a value from itself is zero.
	if CompareExpr(lhs, rhs) == 0 {
		return NewConstantExpr(0, ExprWidth(lhs))
	}

	// Compute constant if both sides are constant.
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.Sub(rhs)
		}
	}

	// Refactor to XOR for boolean expressions.
	if ExprWidth(lhs) == WidthBool {
		return NewBinaryExpr(XOR, lhs, rhs)
	}

	// If constant is on right side, refactor to addition with LHS & RHS flipped.
	if rhs, ok := rhs.(*ConstantExpr); ok && !IsConstantExpr(lhs) {
		return NewBinaryExpr(ADD, NewConstantExpr(0, ExprWidth(rhs)).Sub(rhs), lhs)
	}

	// Combine with children of RHS binary expression, if possible.
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*BinaryExpr); ok {
			if rhs.Op == ADD && IsConstantExpr(rhs.LHS) { // X - (Y+z) == (X-Y) - z
				return NewBinaryExpr(SUB, NewBinaryExpr(SUB, lhs, rhs.LHS), rhs.RHS)
			} else if rhs.Op == SUB && IsConstantExpr(rhs.LHS) { // X - (Y-z) == (X-Y) + z
				return NewBinaryExpr(ADD, NewBinaryExpr(SUB, lhs, rhs.LHS), rhs.RHS)
			}
		}
	}

	// Refactor constant LHS.LHS to a standalone value on LHS.
	if lhs, ok := lhs.(*BinaryExpr); ok && IsConstantExpr(lhs.LHS) {
		if lhs.Op == ADD { // (X+y) - z = X + (y-z)
			return NewBinaryExpr(ADD, lhs.LHS, NewBinaryExpr(SUB, lhs.RHS, rhs))
		} else if lhs.Op == SUB { // (X-y) - z = X - (y+z)
			return NewBinaryExpr(SUB, lhs.LHS, NewBinaryExpr(ADD, lhs.RHS, rhs))
		}
	}

	// Refactor constant RHS.LHS to a standalone value on LHS.
	if rhs, ok := rhs.(*BinaryExpr); ok && IsConstantExpr(rhs.LHS) {
		if rhs.Op == ADD { // x - (Y+z) = (x-z) - Y
			return NewBinaryExpr(SUB, NewBinaryExpr(SUB, lhs, rhs.RHS), rhs.LHS)
		} else if rhs.Op == SUB { // x - (Y-z) = (x+z) - Y
			return NewBinaryExpr(SUB, NewBinaryExpr(ADD, lhs, rhs.RHS), rhs.LHS)
		}
	}

	return &BinaryExpr{Op: SUB, LHS: lhs, RHS: rhs}
}

// newMulExpr returns an expression that represents the product of lhs & rhs.
func newMulExpr(lhs, rhs Expr) Expr {
	// If constant is on right side, swap to left side.
	if IsConstantExpr(rhs) && !IsConstantExpr(lhs) {
		lhs, rhs = rhs, lhs
	}

	// Compute constant if both sides are constant.
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.Mul(rhs)
		}
	}

	// Refactor to XOR for boolean expressions.
	if ExprWidth(lhs) == WidthBool {
		return NewBinaryExpr(AND, lhs, rhs)
	}

	// Optimize for multiplication with a constant 1 or 0.
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if lhs.Value == 1 {
			return rhs
		} else if lhs.Value == 0 {
			return lhs
		}
	}
	return &BinaryExpr{Op: MUL, LHS: lhs, RHS: rhs}
}

// newDivExpr returns an expression that represents the division of lhs & rhs.
func newDivExpr(op BinaryOp, lhs, rhs Expr) Expr {
	assert(op == UDIV || op == SDIV, "invalid div op: %s", op)

	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			if op == UDIV {
				return lhs.UDiv(rhs)
			}
			return lhs.SDiv(rhs)
		}
	}
	if ExprWidth(lhs) == WidthBool {
		return lhs // rhs must be 1
	}
	return &BinaryExpr{Op: op, LHS: lhs, RHS: rhs}
}

// newRemExpr returns an expression that represents the remainder of lhs divided by rhs.
func newRemExpr(op BinaryOp, lhs, rhs Expr) Expr {
	assert(op == UREM || op == SREM, "invalid rem op: %s", op)

	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			if op == UREM {
				return lhs.URem(rhs)
			}
			return lhs.SRem(rhs)
		}
	}
	if ExprWidth(lhs) == WidthBool {
		return NewConstantExpr(0, WidthBool) // rhs must be 1
	}
	return &BinaryExpr{Op: op, LHS: lhs, RHS: rhs}
}

// newAndExpr returns an expression that represents the bitwise AND of lhs & rhs.
func newAndExpr(lhs, rhs Expr) Expr {
	// Compute constant if both sides are constant.
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.And(rhs)
		}
	}

	// If constant is on left side, swap to right side.
	if IsConstantExpr(lhs) && !IsConstantExpr(rhs) {
		lhs, rhs = rhs, lhs
	}

	// Optimize for if constant is all ones or zeros.
	if rhs, ok := rhs.(*ConstantExpr); ok {
		if rhs.IsAllOnes() {
			return lhs
		} else if rhs.Value == 0 {
			return rhs
		}
	}
	return &BinaryExpr{Op: AND, LHS: lhs, RHS: rhs}
}

// newOrExpr returns an expression that represents the bitwise OR of lhs & rhs.
func newOrExpr(lhs, rhs Expr) Expr {
	// Compute constant if both sides are constant.
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.Or(rhs)
		}
	}

	// If constant is on left side, swap to right side.
	if IsConstantExpr(lhs) && !IsConstantExpr(rhs) {
		lhs, rhs = rhs, lhs
	}

	// Optimize for if constant is all ones or zeros.
	if rhs, ok := rhs.(*ConstantExpr); ok {
		if rhs.IsAllOnes() {
			return rhs
		} else if rhs.Value == 0 {
			return lhs
		}
	}
	return &BinaryExpr{Op: OR, LHS: lhs, RHS: rhs}
}

// newXorExpr returns an expression that represents the bitwise XOR of lhs & rhs.
func newXorExpr(lhs, rhs Expr) Expr {
	// If constant is on right side, swap to left side.
	if !IsConstantExpr(lhs) && IsConstantExpr(rhs) {
		lhs, rhs = rhs, lhs
	}

	// Compute constant if both sides are constant.
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if lhs.Value == 0 {
			return rhs
		} else if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.Xor(rhs)
		}
	}

	return &BinaryExpr{Op: XOR, LHS: lhs, RHS: rhs}
}

// newShlExpr returns an expression that represents the shift-left of lhs by rhs bits.
func newShlExpr(lhs, rhs Expr) Expr {
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.Shl(rhs)
		}
	}
	if ExprWidth(lhs) == WidthBool { // l & !r
		return NewBinaryExpr(AND, lhs, NewIsZeroExpr(rhs))
	}
	return &BinaryExpr{Op: SHL, LHS: lhs, RHS: rhs}
}

// newLShrExpr returns an expression that represents the logical shift-right of lhs by rhs bits.
func newLShrExpr(lhs, rhs Expr) Expr {
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.LShr(rhs)
		}
	}
	if ExprWidth(lhs) == WidthBool {
		return NewBinaryExpr(AND, lhs, NewIsZeroExpr(rhs)) // l & !r
	}
	return &BinaryExpr{Op: LSHR, LHS: lhs, RHS: rhs}
}

// newAShrExpr returns an expression that represents the arithmetic shift-right of lhs by rhs bits.
func newAShrExpr(lhs, rhs Expr) Expr {
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.AShr(rhs)
		}
	}
	if ExprWidth(lhs) == WidthBool { // l
		return lhs
	}
	return &BinaryExpr{Op: ASHR, LHS: lhs, RHS: rhs}
}

// newEqExpr returns an expression that represents the equality of lhs and rhs.
func newEqExpr(lhs, rhs Expr) Expr {
	// If constant is on right side, swap to left side.
	if !IsConstantExpr(lhs) && IsConstantExpr(rhs) {
		lhs, rhs = rhs, lhs
	}

	// Compute constant if both sides are constant.
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.Eq(rhs)
		}

		width := ExprWidth(lhs)
		switch rhs := rhs.(type) {
		case *BinaryExpr:
			switch rhs.Op {
			case EQ:
				if width == WidthBool {
					if lhs.IsTrue() {
						return rhs
					} else if IsConstantFalse(lhs) && IsConstantFalse(rhs.LHS) {
						return rhs.RHS // 0 == (0 == A) => A
					}
				}
			case OR:
				if width == WidthBool {
					if lhs.IsTrue() {
						return rhs // T == X || Y => X || Y
					} else if ExprWidth(rhs.LHS) == WidthBool {
						return NewBinaryExpr(AND, NewIsZeroExpr(rhs.LHS), NewIsZeroExpr(rhs.RHS)) // F == X || Y => !X && !Y
					}
				}
			case ADD:
				if IsConstantExpr(rhs.LHS) { // X = Y + z => X - Y = z
					return NewBinaryExpr(EQ, NewBinaryExpr(SUB, lhs, rhs.LHS), rhs.RHS)
				}
			case SUB:
				if IsConstantExpr(rhs.LHS) { // X = Y - z => Y - X = z
					return NewBinaryExpr(EQ, NewBinaryExpr(SUB, rhs.LHS, lhs), rhs.RHS)
				}
			}

		case *CastExpr:
			trunc := lhs.ZExt(ExprWidth(rhs.Src))
			if rhs.Signed { // (sext(a,T)==c) == (a==c)
				if CompareExpr(lhs, trunc.SExt(width)) == 0 {
					return NewBinaryExpr(EQ, rhs.Src, trunc)
				}
				return NewConstantExpr(0, WidthBool)
			} else { // (zext(a,T)==c) == (a==c)
				if CompareExpr(lhs, trunc.ZExt(width)) == 0 {
					return NewBinaryExpr(EQ, rhs.Src, trunc)
				}
				return NewConstantExpr(0, WidthBool)
			}
		}
	}

	if CompareExpr(lhs, rhs) == 0 {
		return NewConstantExpr(1, WidthBool)
	}
	return &BinaryExpr{Op: EQ, LHS: lhs, RHS: rhs}
}

// newUltExpr returns an expression that represents the if lhs is less than rhs (unsigned).
func newUltExpr(lhs, rhs Expr) Expr {
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.Ult(rhs)
		}
	}
	if ExprWidth(lhs) == WidthBool { // !lhs && rhs
		return NewBinaryExpr(AND, NewIsZeroExpr(lhs), rhs)
	}
	return &BinaryExpr{Op: ULT, LHS: lhs, RHS: rhs}
}

// newUltExpr returns an expression that represents the if lhs is less than or equal to rhs (unsigned).
func newUleExpr(lhs, rhs Expr) Expr {
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.Ule(rhs)
		}
	}
	if ExprWidth(lhs) == WidthBool { // !(lhs && !rhs)
		return NewBinaryExpr(OR, NewIsZeroExpr(lhs), rhs)
	}
	return &BinaryExpr{Op: ULE, LHS: lhs, RHS: rhs}
}

// newSltExpr returns an expression that represents the if lhs is less than rhs (signed).
func newSltExpr(lhs, rhs Expr) Expr {
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.Slt(rhs)
		}
	}
	if ExprWidth(lhs) == WidthBool { // lhs && !rhs
		return NewBinaryExpr(AND, lhs, NewIsZeroExpr(rhs))
	}
	return &BinaryExpr{Op: SLT, LHS: lhs, RHS: rhs}
}

// newSleExpr returns an expression that represents the if lhs is less than or equal to rhs (signed).
func newSleExpr(lhs, rhs Expr) Expr {
	if lhs, ok := lhs.(*ConstantExpr); ok {
		if rhs, ok := rhs.(*ConstantExpr); ok {
			return lhs.Sle(rhs)
		}
	}
	if ExprWidth(lhs) == WidthBool { // !(!lhs && rhs)
		return NewBinaryExpr(OR, lhs, NewIsZeroExpr(rhs))
	}
	return &BinaryExpr{Op: SLE, LHS: lhs, RHS: rhs}
}

// SelectExpr represents a one byte read from an array.
type SelectExpr struct {
	Array *Array
	Index Expr
}

// NewSelectExpr returns a new instance of SelectExpr based on a given array.
func NewSelectExpr(a *Array, index Expr) Expr {
	return &SelectExpr{
		Array: a,
		Index: index,
	}
}

// String returns the string representation of the expression.
func (e *SelectExpr) String() string {
	return fmt.Sprintf("(select %s %s)", e.Array, e.Index)
}

// ConcatExpr represents a concatenation of two expressions.
type ConcatExpr struct {
	MSB Expr
	LSB Expr
}

// NewConcatExpr returns a new instance of ConcatExpr.
func NewConcatExpr(msb, lsb Expr) Expr {
	// Combine expressions if they are both constants.
	if msb, ok := msb.(*ConstantExpr); ok {
		if lsb, ok := lsb.(*ConstantExpr); ok {
			return msb.Concat(lsb)
		}
	}

	// Combine extract expressions if they are contiguous.
	if msb, ok := msb.(*ExtractExpr); ok {
		if lsb, ok := lsb.(*ExtractExpr); ok {
			if msb.Expr == lsb.Expr && lsb.Offset+lsb.Width == msb.Offset {
				return NewExtractExpr(msb.Expr, lsb.Offset, msb.Width+lsb.Width)
			}
		}
	}

	return &ConcatExpr{
		MSB: msb,
		LSB: lsb,
	}
}

// String returns the string representation of the expression.
func (e *ConcatExpr) String() string {
	return fmt.Sprintf("(concat %s %s)", e.MSB, e.LSB)
}

// ExtractExpr represents the extraction of a set of bits at a given offset/width.
type ExtractExpr struct {
	Expr   Expr
	Offset uint
	Width  uint
}

// NewExtractExpr returns a new instance of ExtractExpr.
func NewExtractExpr(expr Expr, offset uint, width uint) Expr {
	kw := ExprWidth(expr)
	assert(width > 0, "extract width cannot be zero")
	assert(offset+width <= kw, "extract out of bounds: %d+%d > %d", width, offset, kw)

	if width == kw {
		return expr
	} else if expr, ok := expr.(*ConstantExpr); ok {
		return expr.Extract(offset, width)
	}

	// Extract(Concat)
	if expr, ok := expr.(*ConcatExpr); ok {
		// Directly extract from MSB if we skip over LSB.
		if offset >= ExprWidth(expr.LSB) {
			return NewExtractExpr(expr.MSB, offset-ExprWidth(expr.LSB), width)
		}

		// Directly extract from LSB if we skip over MSB.
		if offset+width <= ExprWidth(expr.LSB) {
			return NewExtractExpr(expr.LSB, offset, width)
		}

		// Convert extraction to a concatenation of two extractions.
		// E(C(x,y)) = C(E(x), E(y))
		return NewConcatExpr(
			NewExtractExpr(expr.MSB, 0, width-ExprWidth(expr.LSB)+offset),
			NewExtractExpr(expr.LSB, offset, ExprWidth(expr.MSB)-offset),
		)
	}

	return &ExtractExpr{
		Expr:   expr,
		Offset: offset,
		Width:  width,
	}
}

// String returns the string representation of the expression.
func (e *ExtractExpr) String() string {
	return fmt.Sprintf("(extract %s %d %d)", e.Expr, e.Offset, e.Width)
}

// NotExpr represents a bitwise not of an expression.
type NotExpr struct {
	Expr Expr
}

// NewNotExpr returns a new instance of NotExpr.
func NewNotExpr(expr Expr) Expr {
	if expr, ok := expr.(*ConstantExpr); ok {
		return expr.Not()
	}
	return &NotExpr{Expr: expr}
}

// String returns the string representation of the expression.
func (e *NotExpr) String() string {
	return fmt.Sprintf("(not %s)", e.Expr)
}

// CastExpr represents an expression that casts an expression to a new width.
type CastExpr struct {
	Src    Expr
	Width  uint
	Signed bool
}

// NewCastExpr returns a new instance of CastExpr.
func NewCastExpr(src Expr, width uint, signed bool) Expr {
	if signed {
		return newSExtExpr(src, width)
	}
	return newZExtExpr(src, width)
}

// newZExtExpr returns a new zero-extension binary operation.
func newZExtExpr(src Expr, w uint) Expr {
	sw := ExprWidth(src)
	if w == sw { // nop
		return src
	} else if w < sw { // truncate
		return NewExtractExpr(src, 0, w)
	} else if src, ok := src.(*ConstantExpr); ok {
		return src.ZExt(w)
	}
	return &CastExpr{Src: src, Width: w, Signed: false}
}

// newZExtExpr returns a new signed-extension binary operation.
func newSExtExpr(src Expr, w uint) Expr {
	sw := ExprWidth(src)
	if w == sw { // nop
		return src
	} else if w < sw { // truncate
		return NewExtractExpr(src, 0, w)
	} else if src, ok := src.(*ConstantExpr); ok {
		return src.SExt(w)
	}
	return &CastExpr{Src: src, Width: w, Signed: true}
}

// String returns the string representation of the expression.
func (e *CastExpr) String() string {
	if e.Signed {
		return fmt.Sprintf("(sext %s %d)", e.Src, e.Width)
	}
	return fmt.Sprintf("(zext %s %d)", e.Src, e.Width)
}

// ConstantExpr represents an arbitrary precision integer.
type ConstantExpr struct {
	Value uint64
	Width uint
}

// NewConstantExpr returns a new instance of ConstantExpr.
func NewConstantExpr(value uint64, width uint) *ConstantExpr {
	return &ConstantExpr{
		Value: value & ((1 << width) - 1),
		Width: width,
	}
}

// NewConstantExpr8 returns a 8-bit constant expression.
func NewConstantExpr8(value uint64) *ConstantExpr {
	return NewConstantExpr(value, 8)
}

// NewConstantExpr16 returns a 16-bit constant expression.
func NewConstantExpr16(value uint64) *ConstantExpr {
	return NewConstantExpr(value, 16)
}

// NewConstantExpr32 returns a 32-bit constant expression.
func NewConstantExpr32(value uint64) *ConstantExpr {
	return NewConstantExpr(value, 32)
}

// NewConstantExpr64 returns a 64-bit constant expression.
func NewConstantExpr64(value uint64) *ConstantExpr {
	return NewConstantExpr(value, 64)
}

// NewBoolConstantExpr is an ease of use function for creating constant boolean expressions.
func NewBoolConstantExpr(value bool) *ConstantExpr {
	if value {
		return &ConstantExpr{Value: 1, Width: WidthBool}
	}
	return &ConstantExpr{Value: 0, Width: WidthBool}
}

// String returns the string representation of the expression.
func (e *ConstantExpr) String() string {
	return fmt.Sprintf("(const %d %d)", e.Value, e.Width)
}

// IsTrue returns true if this is a boolean true expression.
func (e *ConstantExpr) IsTrue() bool {
	return e.Width == WidthBool && e.Value != 0
}

// IsFalse returns true if this is a boolean false expression.
func (e *ConstantExpr) IsFalse() bool {
	return e.Width == WidthBool && e.Value == 0
}

// IsAllOnes returns true if all bits in the value are one.
func (e *ConstantExpr) IsAllOnes() bool {
	return e.Value == bitmask(e.Width)
}

// Add returns the sum of e and other.
func (e *ConstantExpr) Add(other *ConstantExpr) *ConstantExpr {
	assert(e.Width == other.Width, "add: width mismatch: %d != %d", e.Width, other.Width)
	return NewConstantExpr(e.Value+other.Value, e.Width)
}

// Sub returns the difference of e and other.
func (e *ConstantExpr) Sub(other *ConstantExpr) *ConstantExpr {
	assert(e.Width == other.Width, "sub: width mismatch: %d != %d", e.Width, other.Width)
	return NewConstantExpr(e.Value-other.Value, e.Width)
}

// Mul returns the product of e and other.
func (e *ConstantExpr) Mul(other *ConstantExpr) *ConstantExpr {
	assert(e.Width == other.Width, "mul: width mismatch: %d != %d", e.Width, other.Width)
	return NewConstantExpr((e.Value*other.Value)&bitmask(e.Width), e.Width)
}

// URem returns the quotient of unsigned division of e and other.
func (e *ConstantExpr) UDiv(other *ConstantExpr) *ConstantExpr {
	assert(e.Width == other.Width, "udiv: width mismatch: %d != %d", e.Width, other.Width)
	switch e.Width {
	case Width8:
		return NewConstantExpr(uint64(uint8(e.Value)/uint8(other.Value)), e.Width)
	case Width16:
		return NewConstantExpr(uint64(uint16(e.Value)/uint16(other.Value)), e.Width)
	case Width32:
		return NewConstantExpr(uint64(uint32(e.Value)/uint32(other.Value)), e.Width)
	case Width64:
		return NewConstantExpr(uint64(uint64(e.Value)/uint64(other.Value)), e.Width)
	default:
		panic(fmt.Sprintf("udiv: non-standard width: %d", e.Width))
	}
}

// URem returns the quotient of signed division of e and other.
func (e *ConstantExpr) SDiv(other *ConstantExpr) *ConstantExpr {
	assert(e.Width == other.Width, "sdiv: width mismatch: %d != %d", e.Width, other.Width)
	switch e.Width {
	case Width8:
		return NewConstantExpr(uint64(int8(e.Value)/int8(other.Value)), e.Width)
	case Width16:
		return NewConstantExpr(uint64(int16(e.Value)/int16(other.Value)), e.Width)
	case Width32:
		return NewConstantExpr(uint64(int32(e.Value)/int32(other.Value)), e.Width)
	case Width64:
		return NewConstantExpr(uint64(int64(e.Value)/int64(other.Value)), e.Width)
	default:
		panic(fmt.Sprintf("sdiv: non-standard width: %d", e.Width))
	}
}

// URem returns the remainder of unsigned division of e and other.
func (e *ConstantExpr) URem(other *ConstantExpr) *ConstantExpr {
	assert(e.Width == other.Width, "urem: width mismatch: %d != %d", e.Width, other.Width)
	switch e.Width {
	case Width8:
		return NewConstantExpr(uint64(uint8(e.Value)%uint8(other.Value)), e.Width)
	case Width16:
		return NewConstantExpr(uint64(uint16(e.Value)%uint16(other.Value)), e.Width)
	case Width32:
		return NewConstantExpr(uint64(uint32(e.Value)%uint32(other.Value)), e.Width)
	case Width64:
		return NewConstantExpr(uint64(uint64(e.Value)%uint64(other.Value)), e.Width)
	default:
		panic(fmt.Sprintf("urem: non-standard width: %d", e.Width))
	}
}

// SRem returns the remainder of signed division of e and other.
func (e *ConstantExpr) SRem(other *ConstantExpr) *ConstantExpr {
	assert(e.Width == other.Width, "srem: width mismatch: %d != %d", e.Width, other.Width)
	switch e.Width {
	case Width8:
		return NewConstantExpr(uint64(int8(e.Value)%int8(other.Value)), e.Width)
	case Width16:
		return NewConstantExpr(uint64(int16(e.Value)%int16(other.Value)), e.Width)
	case Width32:
		return NewConstantExpr(uint64(int32(e.Value)%int32(other.Value)), e.Width)
	case Width64:
		return NewConstantExpr(uint64(int64(e.Value)%int64(other.Value)), e.Width)
	default:
		panic(fmt.Sprintf("srem: non-standard width: %d", e.Width))
	}
}

// And returns the bitwise AND of e and other.
func (e *ConstantExpr) And(other *ConstantExpr) *ConstantExpr {
	assert(e.Width == other.Width, "and: width mismatch: %d != %d", e.Width, other.Width)
	return NewConstantExpr(e.Value&other.Value, e.Width)
}

// Or returns the bitwise OR of e and other.
func (e *ConstantExpr) Or(other *ConstantExpr) *ConstantExpr {
	assert(e.Width == other.Width, "or: width mismatch: %d != %d", e.Width, other.Width)
	return NewConstantExpr(e.Value|other.Value, e.Width)
}

// Xor returns the bitwise XOR of e and other.
func (e *ConstantExpr) Xor(other *ConstantExpr) *ConstantExpr {
	assert(e.Width == other.Width, "xor: width mismatch: %d != %d", e.Width, other.Width)
	return NewConstantExpr(e.Value^other.Value, e.Width)
}

// Shl returns the value of e shifted left by other number of bits.
func (e *ConstantExpr) Shl(other *ConstantExpr) *ConstantExpr {
	switch e.Width {
	case Width8:
		return NewConstantExpr(uint64(uint8(e.Value)<<other.Value), e.Width)
	case Width16:
		return NewConstantExpr(uint64(uint16(e.Value)<<other.Value), e.Width)
	case Width32:
		return NewConstantExpr(uint64(uint32(e.Value)<<other.Value), e.Width)
	case Width64:
		return NewConstantExpr(uint64(e.Value)<<other.Value, e.Width)
	default:
		panic("shl: non-standard width")
	}
}

// LShr returns the value of e logically shifted right by other number of bits.
func (e *ConstantExpr) LShr(other *ConstantExpr) *ConstantExpr {
	switch e.Width {
	case Width8:
		return NewConstantExpr(uint64(uint8(e.Value)>>other.Value), e.Width)
	case Width16:
		return NewConstantExpr(uint64(uint16(e.Value)>>other.Value), e.Width)
	case Width32:
		return NewConstantExpr(uint64(uint32(e.Value)>>other.Value), e.Width)
	case Width64:
		return NewConstantExpr(uint64(e.Value)>>other.Value, e.Width)
	default:
		panic("lshr: non-standard width")
	}
}

// AShr returns the value of e arithmetically shifted right by other number of bits.
func (e *ConstantExpr) AShr(other *ConstantExpr) *ConstantExpr {
	switch e.Width {
	case Width8:
		return NewConstantExpr(uint64(uint8(int8(e.Value)>>other.Value)), e.Width)
	case Width16:
		return NewConstantExpr(uint64(uint16(int16(e.Value)>>other.Value)), e.Width)
	case Width32:
		return NewConstantExpr(uint64(uint32(int32(e.Value)>>other.Value)), e.Width)
	case Width64:
		return NewConstantExpr(uint64(int64(e.Value)>>other.Value), e.Width)
	default:
		panic("ashr: non-standard width")
	}
}

// Eq returns the equality of e and other.
func (e *ConstantExpr) Eq(other *ConstantExpr) *ConstantExpr {
	assert(e.Width == other.Width, "eq: width mismatch: %d != %d", e.Width, other.Width)
	if e.Value == other.Value {
		return NewConstantExpr(1, WidthBool)
	}
	return NewConstantExpr(0, WidthBool)
}

// Ult returns the unsigned less than comparison of e to other.
func (e *ConstantExpr) Ult(other *ConstantExpr) *ConstantExpr {
	switch e.Width {
	case Width8:
		return NewBoolConstantExpr(uint8(e.Value) < uint8(other.Value))
	case Width16:
		return NewBoolConstantExpr(uint16(e.Value) < uint16(other.Value))
	case Width32:
		return NewBoolConstantExpr(uint32(e.Value) < uint32(other.Value))
	case Width64:
		return NewBoolConstantExpr(uint64(e.Value) < uint64(other.Value))
	default:
		panic("ult: non-standard width")
	}
}

// Ugt returns the unsigned greater than comparison of e to other.
func (e *ConstantExpr) Ugt(other *ConstantExpr) *ConstantExpr {
	return other.Ult(e)
}

// Ule returns the unsigned less than or equal to comparison of e to other.
func (e *ConstantExpr) Ule(other *ConstantExpr) *ConstantExpr {
	switch e.Width {
	case Width8:
		return NewBoolConstantExpr(uint8(e.Value) <= uint8(other.Value))
	case Width16:
		return NewBoolConstantExpr(uint16(e.Value) <= uint16(other.Value))
	case Width32:
		return NewBoolConstantExpr(uint32(e.Value) <= uint32(other.Value))
	case Width64:
		return NewBoolConstantExpr(uint64(e.Value) <= uint64(other.Value))
	default:
		panic("ule: non-standard width")
	}
}

// Uge returns the unsigned greater than or equal to comparison of e to other.
func (e *ConstantExpr) Uge(other *ConstantExpr) *ConstantExpr {
	return other.Ule(e)
}

// Slt returns the signed less than comparison of e to other.
func (e *ConstantExpr) Slt(other *ConstantExpr) *ConstantExpr {
	switch e.Width {
	case Width8:
		return NewBoolConstantExpr(int8(e.Value) < int8(other.Value))
	case Width16:
		return NewBoolConstantExpr(int16(e.Value) < int16(other.Value))
	case Width32:
		return NewBoolConstantExpr(int32(e.Value) < int32(other.Value))
	case Width64:
		return NewBoolConstantExpr(int64(e.Value) < int64(other.Value))
	default:
		panic("slt: non-standard width")
	}
}

// Sgt returns the signed greater than comparison of e to other.
func (e *ConstantExpr) Sgt(other *ConstantExpr) *ConstantExpr {
	return other.Slt(e)
}

// Sle returns the signed less than or equal to comparison of e to other.
func (e *ConstantExpr) Sle(other *ConstantExpr) *ConstantExpr {
	switch e.Width {
	case Width8:
		return NewBoolConstantExpr(int8(e.Value) <= int8(other.Value))
	case Width16:
		return NewBoolConstantExpr(int16(e.Value) <= int16(other.Value))
	case Width32:
		return NewBoolConstantExpr(int32(e.Value) <= int32(other.Value))
	case Width64:
		return NewBoolConstantExpr(int64(e.Value) <= int64(other.Value))
	default:
		panic("sle: non-standard width")
	}
}

// Sge returns the signed greater than or equal to comparison of e to other.
func (e *ConstantExpr) Sge(other *ConstantExpr) *ConstantExpr {
	return other.Sle(e)
}

// ZExt returns the zero-extension of e to a new width.
func (e *ConstantExpr) ZExt(width uint) *ConstantExpr {
	if e.Width == width {
		return e
	} else if width == WidthBool {
		return NewBoolConstantExpr(e.Value != 0)
	}
	return NewConstantExpr(e.Value, width)
}

// ZExt returns the sign-extension of e to a new width.
func (e *ConstantExpr) SExt(width uint) *ConstantExpr {
	if e.Width == width {
		return e
	}

	switch width {
	case Width8:
		switch e.Width {
		case Width16:
			return NewConstantExpr(uint64(int16(int8(e.Value))), width)
		case Width32:
			return NewConstantExpr(uint64(int32(int8(e.Value))), width)
		case Width64:
			return NewConstantExpr(uint64(int64(int8(e.Value))), width)
		}
	case Width16:
		switch e.Width {
		case Width8:
			return NewConstantExpr(uint64(int8(int16(e.Value))), width)
		case Width32:
			return NewConstantExpr(uint64(int32(int16(e.Value))), width)
		case Width64:
			return NewConstantExpr(uint64(int64(int16(e.Value))), width)
		}
	case Width32:
		switch e.Width {
		case Width8:
			return NewConstantExpr(uint64(int8(int32(e.Value))), width)
		case Width16:
			return NewConstantExpr(uint64(int16(int32(e.Value))), width)
		case Width64:
			return NewConstantExpr(uint64(int64(int32(e.Value))), width)
		}
	case Width64:
		switch e.Width {
		case Width8:
			return NewConstantExpr(uint64(int8(int64(e.Value))), width)
		case Width16:
			return NewConstantExpr(uint64(int16(int64(e.Value))), width)
		case Width32:
			return NewConstantExpr(uint64(int32(int64(e.Value))), width)
		}
	}
	panic(fmt.Sprintf("sext: non-standard width: %d -> %d", e.Width, width))
}

// Not returns the bitwise NOT of the expression.
func (e *ConstantExpr) Not() *ConstantExpr {
	return NewConstantExpr((^e.Value)&bitmask(e.Width), e.Width)
}

// Extract returns width number of bits starting at offset.
func (e *ConstantExpr) Extract(offset, width uint) *ConstantExpr {
	return NewConstantExpr(uint64(int64(e.Value)>>offset)&bitmask(e.Width), width)
}

// Concat returns the concatenation of e and lsb.
func (e *ConstantExpr) Concat(lsb *ConstantExpr) *ConstantExpr {
	return NewConstantExpr((e.Value<<lsb.Width)|lsb.Value, ExprWidth(e)+ExprWidth(lsb))
}

func bitmask(width uint) uint64 {
	return (1 << width) - 1
}

// IsConstantExpr returns true if expr is an instance of ConstantExpr.
func IsConstantExpr(expr Expr) bool {
	_, ok := expr.(*ConstantExpr)
	return ok
}

// IsConstantTrue returns true if expr is an instance of ConstantExpr and is true.
func IsConstantTrue(expr Expr) bool {
	tmp, ok := expr.(*ConstantExpr)
	return ok && tmp.IsTrue()
}

// IsConstantFalse returns true if expr is an instance of ConstantExpr and is false.
func IsConstantFalse(expr Expr) bool {
	tmp, ok := expr.(*ConstantExpr)
	return ok && tmp.IsFalse()
}

// NewIsZeroExpr returns an expression that checks the equality of other to zero.
func NewIsZeroExpr(other Expr) Expr {
	return NewBinaryExpr(EQ, other, NewConstantExpr(0, ExprWidth(other)))
}

type NotOptimizedExpr struct {
	Src Expr
}

// NewNotOptimizedExpr returns a new instance of NotOptimizedExpr.
func NewNotOptimizedExpr(src Expr) Expr {
	return &NotOptimizedExpr{Src: src}
}

// String returns the string representation of the expression.
func (e *NotOptimizedExpr) String() string {
	return fmt.Sprintf("(no-opt %s)", e.Src)
}

// Tuple represents a slice of bindings.
type Tuple []Binding

// String returns the string representation of the tuple.
func (a Tuple) String() string {
	var buf bytes.Buffer
	buf.WriteRune('[')
	for i := range a {
		buf.WriteString(a[i].String())
		if i < len(a)-1 {
			buf.WriteRune(' ')
		}
	}
	buf.WriteRune(']')
	return buf.String()
}

// CompareExpr returns an integer comparing two expressions.
// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func CompareExpr(a, b Expr) int {
	if a == nil && b != nil {
		return -1
	} else if a != nil && b == nil {
		return 1
	} else if a == nil && b == nil {
		return 0
	}

	if ak, bk := exprKind(a), exprKind(b); ak < bk {
		return -1
	} else if ak > bk {
		return 1
	}

	switch a := a.(type) {
	case *ConstantExpr:
		return compareConstantExpr(a, b.(*ConstantExpr))
	case *NotOptimizedExpr:
		return compareNotOptimizedExpr(a, b.(*NotOptimizedExpr))
	case *SelectExpr:
		return compareSelectExpr(a, b.(*SelectExpr))
	case *ConcatExpr:
		return compareConcatExpr(a, b.(*ConcatExpr))
	case *ExtractExpr:
		return compareExtractExpr(a, b.(*ExtractExpr))
	case *NotExpr:
		return compareNotExpr(a, b.(*NotExpr))
	case *CastExpr:
		return compareCastExpr(a, b.(*CastExpr))
	case *BinaryExpr:
		return compareBinaryExpr(a, b.(*BinaryExpr))
	default:
		panic("unreachable")
	}
}

func compareConstantExpr(a, b *ConstantExpr) int {
	if a.Width < b.Width {
		return -1
	} else if a.Width > b.Width {
		return 1
	}

	if a.Value < b.Value {
		return -1
	} else if a.Value > b.Value {
		return 1
	}
	return 0
}

func compareNotOptimizedExpr(a, b *NotOptimizedExpr) int {
	return CompareExpr(a.Src, b.Src)
}

func compareSelectExpr(a, b *SelectExpr) int {
	if cmp := CompareExpr(a.Index, b.Index); cmp != 0 {
		return cmp
	}
	return CompareArray(a.Array, b.Array)
}

func compareConcatExpr(a, b *ConcatExpr) int {
	if cmp := CompareExpr(a.MSB, b.MSB); cmp != 0 {
		return cmp
	}
	return CompareExpr(a.LSB, b.LSB)
}

func compareExtractExpr(a, b *ExtractExpr) int {
	if a.Offset < b.Offset {
		return -1
	} else if a.Offset > b.Offset {
		return 1
	}

	if a.Width < b.Width {
		return -1
	} else if a.Width > b.Width {
		return 1
	}
	return CompareExpr(a.Expr, b.Expr)
}

func compareNotExpr(a, b *NotExpr) int {
	return CompareExpr(a.Expr, b.Expr)
}

func compareCastExpr(a, b *CastExpr) int {
	if a.Signed && !b.Signed {
		return -1
	} else if !a.Signed && b.Signed {
		return 1
	}

	if a.Width < b.Width {
		return -1
	} else if a.Width > b.Width {
		return 1
	}
	return CompareExpr(a.Src, b.Src)
}

func compareBinaryExpr(a, b *BinaryExpr) int {
	if a.Op < b.Op {
		return -1
	} else if a.Op > b.Op {
		return 1
	}
	if cmp := CompareExpr(a.LHS, b.LHS); cmp != 0 {
		return cmp
	}
	return CompareExpr(a.RHS, b.RHS)
}

// exprKind returns a numeric value for the type of expression.
// Only used internally for equality checks and sorting.
func exprKind(expr Expr) int {
	switch expr.(type) {
	case *ConstantExpr:
		return 1
	case *NotOptimizedExpr:
		return 2
	case *SelectExpr:
		return 3
	case *ConcatExpr:
		return 4
	case *ExtractExpr:
		return 5
	case *NotExpr:
		return 6
	case *CastExpr:
		return 7
	case *BinaryExpr:
		return 8
	default:
		panic("unreachable")
	}
}

// ExprVisitor represents a visitor that can be passed to WalkExpr().
type ExprVisitor interface {
	// Executed for every visited node. Return a different expression to replace it.
	Visit(expr Expr) (Expr, ExprVisitor)
}

func WalkExpr(v ExprVisitor, expr Expr) Expr {
	other, v := v.Visit(expr)
	if v == nil {
		return other
	}

	switch expr := expr.(type) {
	case *BinaryExpr:
		if other := WalkExpr(v, expr.LHS); other != expr.LHS {
			expr.LHS = other
		}
		if other := WalkExpr(v, expr.RHS); other != expr.RHS {
			expr.RHS = other
		}
	case *CastExpr:
		if other := WalkExpr(v, expr.Src); other != expr.Src {
			expr.Src = other
		}
	case *ConcatExpr:
		if other := WalkExpr(v, expr.MSB); other != expr.MSB {
			expr.MSB = other
		}
		if other := WalkExpr(v, expr.LSB); other != expr.LSB {
			expr.LSB = other
		}
	case *ConstantExpr:
		// nop
	case *ExtractExpr:
		if other := WalkExpr(v, expr.Expr); other != expr.Expr {
			expr.Expr = other
		}
	case *NotExpr:
		if other := WalkExpr(v, expr.Expr); other != expr.Expr {
			expr.Expr = other
		}
	case *NotOptimizedExpr:
		if other := WalkExpr(v, expr.Src); other != expr.Src {
			expr.Src = other
		}
	case *SelectExpr:
		if other := WalkExpr(v, expr.Index); other != expr.Index {
			expr.Index = other
		}
		for upd := expr.Array.Updates; upd != nil; upd = upd.Next {
			if other := WalkExpr(v, upd.Index); other != upd.Index {
				upd.Index = other
			}
			if other := WalkExpr(v, upd.Value); other != upd.Value {
				upd.Value = other
			}
		}
	default:
		panic("unreachable")
	}

	return other
}

// FindArrays returns all symbolic arrays in the expression tree.
func FindArrays(exprs ...Expr) []*Array {
	v := newArrayExprVisitor()
	for _, expr := range exprs {
		WalkExpr(v, expr)
	}

	a := make([]*Array, 0, len(v.m))
	for _, array := range v.m {
		a = append(a, array)
	}
	sort.Slice(a, func(i, j int) bool { return CompareArray(a[i], a[j]) == -1 })

	return a
}

type arrayExprVisitor struct {
	m map[uint64]*Array
}

func newArrayExprVisitor() *arrayExprVisitor {
	return &arrayExprVisitor{m: make(map[uint64]*Array)}
}

func (v *arrayExprVisitor) Visit(expr Expr) (Expr, ExprVisitor) {
	if expr, ok := expr.(*SelectExpr); ok && expr.Array.IsSymbolic() {
		if _, ok := v.m[expr.Array.ID]; !ok {
			v.m[expr.Array.ID] = expr.Array
		}
	}
	return expr, v
}

// ExprEvaluator evaluates expressions using known array values.
type ExprEvaluator struct {
	m map[uint64][]byte // mapping of array id to value
}

// NewExprEvaluator returns a new instance of ExprEvaluator with the given array/value mapping.
func NewExprEvaluator(arrays []*Array, values [][]byte) *ExprEvaluator {
	assert(len(arrays) == len(values), "array/value count mismatch: %d != %d", len(arrays), len(values))

	m := make(map[uint64][]byte)
	for i, array := range arrays {
		_, ok := m[array.ID]
		assert(!ok, "duplicate array: id=%d", array.ID)
		m[array.ID] = values[i]
	}

	return &ExprEvaluator{m: m}
}

// Evaluate evaluates expr to a constant expression.
// Returns an error if an unknown array is encountered.
func (ee *ExprEvaluator) Evaluate(expr Expr) (*ConstantExpr, error) {
	switch expr := expr.(type) {
	case *BinaryExpr:
		lhs, err := ee.Evaluate(expr.LHS)
		if err != nil {
			return nil, err
		}
		rhs, err := ee.Evaluate(expr.RHS)
		if err != nil {
			return nil, err
		}
		return NewBinaryExpr(expr.Op, lhs, rhs).(*ConstantExpr), nil
	case *CastExpr:
		src, err := ee.Evaluate(expr.Src)
		if err != nil {
			return nil, err
		}
		return NewCastExpr(src, expr.Width, expr.Signed).(*ConstantExpr), nil
	case *ConcatExpr:
		msb, err := ee.Evaluate(expr.MSB)
		if err != nil {
			return nil, err
		}
		lsb, err := ee.Evaluate(expr.LSB)
		if err != nil {
			return nil, err
		}
		return NewConcatExpr(msb, lsb).(*ConstantExpr), nil
	case *ConstantExpr:
		return expr, nil
	case *ExtractExpr:
		exp, err := ee.Evaluate(expr.Expr)
		if err != nil {
			return nil, err
		}
		return NewExtractExpr(exp, expr.Offset, expr.Width).(*ConstantExpr), nil
	case *NotExpr:
		exp, err := ee.Evaluate(expr.Expr)
		if err != nil {
			return nil, err
		}
		return NewNotExpr(exp).(*ConstantExpr), nil
	case *NotOptimizedExpr:
		return ee.Evaluate(expr.Src)
	case *SelectExpr:
		i, err := ee.Evaluate(expr.Index)
		if err != nil {
			return nil, err
		}

		// Return most recent update to given index, if available.
		for upd := expr.Array.Updates; upd != nil; upd = upd.Next {
			index, err := ee.Evaluate(upd.Index)
			if err != nil {
				return nil, err
			} else if index.Value != i.Value {
				continue
			}
			return ee.Evaluate(upd.Value)
		}

		// Otherwise return original value.
		initial, ok := ee.m[expr.Array.ID]
		if !ok {
			return nil, fmt.Errorf("array not bound: id=%d", expr.Array.ID)
		} else if int(i.Value) >= len(initial) {
			return nil, fmt.Errorf("select index out of bounds: %d >= %d", i.Value, len(initial))
		}
		return NewConstantExpr(uint64(initial[i.Value]), 8), nil

	default:
		return nil, fmt.Errorf("invalid expression type: %T", expr)
	}
}

// minBytes returns smallest number of bytes in which the w fits.
func minBytes(bits uint) uint {
	return (bits + 7) / 8
}
