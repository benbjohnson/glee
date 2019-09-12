package glee

import (
	"fmt"
)

// Array represents an array of symbolic or concrete bytes.
type Array struct {
	ID      uint64       // unique id
	Size    uint         // width, in bytes
	Updates *ArrayUpdate // linked list of symbolic updates
}

// NewArray returns a new Array of the given size.
func NewArray(id uint64, size uint) *Array {
	return &Array{
		ID:   id,
		Size: size,
	}
}

// String returns a string representation of the array.
func (a *Array) String() string {
	if a.ID != 0 {
		return fmt.Sprintf("(array #%d %d)", a.ID, a.Size)
	}
	return fmt.Sprintf("(array %d)", a.Size)
}

// Clone returns a copy of the array.
func (a *Array) Clone() *Array {
	return &Array{
		ID:      a.ID,
		Size:    a.Size,
		Updates: a.Updates,
	}
}

// zero initializes all bytes to zero in-place. Panic if updates already exist.
func (a *Array) zero() {
	assert(a.Updates == nil, "glee.Array: cannot zero-initialize array with updates")
	for i := uint((0)); i < a.Size; i++ {
		a.storeByte(NewConstantExpr64(uint64(i)), NewConstantExpr(0, 8))
	}
}

// Select reads a value from the array.
func (a *Array) Select(offset Expr, width uint, isLittleEndian bool) Expr {
	assert(width > 0, "select: invalid width")

	offset = newZExtExpr(offset, Width64)

	if width == WidthBool {
		return NewExtractExpr(a.selectByte(offset), 0, WidthBool)
	}

	// Handle read byte-by-byte.
	var result Expr
	for i, n := uint64(0), uint64(width)/8; i != n; i++ {
		byteOffset := i
		if !isLittleEndian {
			byteOffset = (n - i - 1)
		}

		value := a.selectByte(NewBinaryExpr(ADD, offset, NewConstantExpr64(byteOffset)))
		if i == 0 {
			result = value
		} else {
			result = NewConcatExpr(value, result)
		}
	}
	return result
}

// selectByte reads a single byte from the array.
//
// Attempts to find a concrete value by traversing the array update history.
// Falls back to a select expression if either the selected index or an update's
// index is symbolic.
func (a *Array) selectByte(index Expr) Expr {
	assert(ExprWidth(index) == 64, "selectByte: invalid array index width: %d", ExprWidth(index))
	for upd := a.Updates; upd != nil; upd = upd.Next {
		cond, ok := NewBinaryExpr(EQ, index, upd.Index).(*ConstantExpr)
		if !ok {
			break // found symbolic index, exit
		} else if cond.IsTrue() {
			return upd.Value
		}
	}
	return NewSelectExpr(a, index)
}

// Store writes a value at an offset. Returns a new copy of the array.
func (a *Array) Store(offset, value Expr, isLittleEndian bool) *Array {
	other := a.Clone()

	offset = newZExtExpr(offset, Width64)

	// Treat bool specially, it is the only non-byte sized write we allow.
	width := ExprWidth(value)
	assert(width > 0, "store: invalid width")
	if width == WidthBool {
		other.storeByte(offset, value)
		return other
	}

	// Otherwise, follow the slow general case.
	for i, n := uint64(0), uint64(width)/8; i != n; i++ {
		byteOffset := i
		if !isLittleEndian {
			byteOffset = (n - i - 1)
		}

		other.storeByte(NewBinaryExpr(ADD, offset, NewConstantExpr64(uint64(byteOffset))), NewExtractExpr(value, uint(i*8), Width8))
	}
	return other
}

// storeByte writes a single byte to the array.
func (a *Array) storeByte(index, value Expr) {
	assert(ExprWidth(index) == 64, "storeByte: invalid array index width: %d", ExprWidth(index))

	// Verify constant is not out of bounds.
	if index, ok := index.(*ConstantExpr); ok {
		assert(index.Value < uint64(a.Size), "storeByte: index out of bounds: %d < %d", index.Value, a.Size)
	}

	// Add update to the head of the chain.
	a.Updates = NewArrayUpdate(index, value, a.Updates)

	// Remove any previous updates to the index from the chain.
	if index, ok := index.(*ConstantExpr); ok {
		prev := a.Updates
		for upd := prev.Next; upd != nil; upd = upd.Next {
			if updIndex, ok := upd.Index.(*ConstantExpr); !ok {
				break // symbolic index
			} else if index.Value == updIndex.Value {
				prev.Next = upd.Next // matching index, remove
			} else {
				prev = upd // no matching index, continue
			}
		}
	}
}

// IsSymbolic returns true if any bytes in the array are symbolic.
func (a *Array) IsSymbolic() bool {
	// Mark all bytes with concrete values.
	bytes := make([]bool, a.Size)
	for upd := a.Updates; upd != nil; upd = upd.Next {
		if index, ok := upd.Index.(*ConstantExpr); !ok {
			return true // found symbolic index
		} else if _, ok := upd.Value.(*ConstantExpr); ok {
			bytes[index.Value] = true // index & value are concrete
		}
	}

	for _, isConcrete := range bytes {
		if !isConcrete {
			return true
		}
	}
	return false
}

// Equal returns a boolean expression stating if a is equal to other.
func (a *Array) Equal(other *Array) Expr {
	// Length is known at runtime so verify first.
	if a.Size != other.Size {
		return NewBoolConstantExpr(false)
	} else if a.Size == 0 {
		return NewBoolConstantExpr(true)
	}

	// Check equality for every byte.
	// Exit early if any concrete byte is unequal.
	var cond Expr
	for i := uint(0); i < a.Size; i++ {
		// Select one at index from each array.
		index := NewConstantExpr64(uint64(i))
		x, y := a.selectByte(index), other.selectByte(index)

		// Compare bytes, exit if known false.
		expr := newEqExpr(x, y)
		if IsConstantFalse(expr) {
			return NewBoolConstantExpr(false)
		}

		// Initialize or join to existing constraint set.
		if i == 0 {
			cond = expr
		} else {
			cond = newAndExpr(cond, expr)
		}
	}
	return cond
}

// NotEqual returns a boolean expression stating if a is not equal to other.
func (a *Array) NotEqual(other *Array) Expr {
	// Length is known at runtime so verify first.
	if a.Size != other.Size {
		return NewBoolConstantExpr(true)
	} else if a.Size == 0 {
		return NewBoolConstantExpr(false)
	}

	// Check inequality for every byte.
	// Exit early if any concrete byte is unequal.
	var cond Expr
	for i := uint(0); i < a.Size; i++ {
		// Select one at index from each array.
		index := NewConstantExpr64(uint64(i))
		x, y := a.selectByte(index), other.selectByte(index)

		// Compare bytes, exit if known inequality.
		expr := NewNotExpr(newEqExpr(x, y))
		if IsConstantTrue(expr) {
			return NewBoolConstantExpr(true)
		}

		// Initialize or join to existing constraint set.
		if i == 0 {
			cond = expr
		} else {
			cond = newOrExpr(cond, expr)
		}
	}
	return cond
}

// CompareArray returns an integer comparing two arrays.
// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func CompareArray(a, b *Array) int {
	if a == nil && b != nil {
		return -1
	} else if a != nil && b == nil {
		return 1
	} else if a == nil && b == nil {
		return 0
	}

	if a.ID < b.ID {
		return -1
	} else if a.ID > b.ID {
		return 1
	}

	if a.Size < b.Size {
		return -1
	} else if a.Size > b.Size {
		return 1
	}

	return CompareArrayUpdate(a.Updates, b.Updates)
}

// ArrayUpdate represents a symbolic update to an array.
type ArrayUpdate struct {
	Index Expr // byte index of update
	Value Expr // byte value to update

	Next *ArrayUpdate // linked list of next update
}

// NewArrayUpdate returns a new instance of ArrayUpdate.
func NewArrayUpdate(index, value Expr, next *ArrayUpdate) *ArrayUpdate {
	return &ArrayUpdate{
		Index: newZExtExpr(index, Width64),
		Value: newZExtExpr(value, Width8),
		Next:  next,
	}
}

// CompareArrayUpdate returns an integer comparing two array updates.
// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func CompareArrayUpdate(a, b *ArrayUpdate) int {
	if a == nil && b != nil {
		return -1
	} else if a != nil && b == nil {
		return 1
	} else if a == nil && b == nil {
		return 0
	}

	if cmp := CompareExpr(a.Index, b.Index); cmp != 0 {
		return cmp
	} else if cmp := CompareExpr(a.Value, b.Value); cmp != 0 {
		return cmp
	}
	return CompareArrayUpdate(a.Next, b.Next)
}
