package glee

import (
	"bytes"
	"errors"
	"fmt"
	"go/constant"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"github.com/benbjohnson/immutable"
	"golang.org/x/tools/go/ssa"
)

// ExecutionState representing a path under exploration.
type ExecutionState struct {
	id int

	// Executor this is executed within.
	executor *Executor

	// Execution hierarchy.
	parent   *ExecutionState
	children []*ExecutionState

	// Call stack
	stack []*StackFrame

	// Shows whether state is running, finished, or terminated by error state.
	status ExecutionStatus
	reason string

	// Heap memory address space.
	heap *immutable.SortedMap

	// Constraints collected so far during execution.
	constraints []Expr

	// Line coverage
	covered map[string]map[uint]struct{}
}

func NewExecutionState(executor *Executor, fn *ssa.Function) *ExecutionState {
	s := &ExecutionState{
		executor: executor,
		status:   ExecutionStatusRunning,
		heap:     immutable.NewSortedMap(&uint64Comparer{}),
	}
	s.Push(fn)
	return s
}

// ID returns an autoincrementing ID assigned by the executor.
func (s *ExecutionState) ID() int { return s.id }

// Executor returns the parent executor of this state.
func (s *ExecutionState) Executor() *Executor {
	return s.executor
}

func (s *ExecutionState) Constraints() []Expr {
	return s.constraints
}

// Clone returns a copy of the state and including deep copies of the stack
// and constraints. However, this does not clone child states.
func (s *ExecutionState) Clone() *ExecutionState {
	stack := make([]*StackFrame, len(s.stack))
	for i := range s.stack {
		stack[i] = s.stack[i].Clone()
	}

	constraints := make([]Expr, len(s.constraints))
	for i := range s.constraints {
		constraints[i] = s.constraints[i]
	}

	return &ExecutionState{
		executor:    s.executor,
		parent:      s.parent,
		status:      s.status,
		heap:        s.heap,
		stack:       stack,
		constraints: constraints,
		covered:     make(map[string]map[uint]struct{}),
	}
}

// Status returns the current status of the state.
// See Reason() for additional information if status is in an error state.
func (s *ExecutionState) Status() ExecutionStatus {
	return s.status
}

// Reason returns additional information about the status of the state.
func (s *ExecutionState) Reason() string {
	return s.reason
}

// Terminated returns true if the state completes execution of a path.
func (s *ExecutionState) Terminated() bool {
	return s.status != ExecutionStatusRunning
}

// Position returns the position of the current instruction in the current file set.
func (s *ExecutionState) Position() token.Position {
	instr := s.Instr()
	if instr == nil {
		return token.Position{}
	}
	switch instr := instr.(type) {
	case *ssa.If:
		return s.executor.prog.Fset.Position(instr.Cond.Pos())
	default:
		return s.executor.prog.Fset.Position(instr.Pos())
	}
}

// Frame returns the current stack frame.
func (s *ExecutionState) Frame() *StackFrame {
	if len(s.stack) == 0 {
		return nil
	}
	return s.stack[len(s.stack)-1]
}

// CallerFrame returns the parent of the current stack frame.
func (s *ExecutionState) CallerFrame() *StackFrame {
	if len(s.stack) <= 1 {
		return nil
	}
	return s.stack[len(s.stack)-2]
}

// Instr returns the current SSA instruction.
func (s *ExecutionState) Instr() ssa.Instruction {
	if frame := s.Frame(); frame != nil {
		return frame.Instr()
	}
	return nil
}

// Eval returns the expression or slice of expression bound to a given SSA value.
func (s *ExecutionState) Eval(value ssa.Value) Binding {
	switch value := value.(type) {
	case *ssa.Const:
		if value.Value == nil {
			size := s.executor.Sizeof(deref(value.Type())) / 8
			_, array := s.Alloc(size)
			array.zero()
			return array
		}

		switch value.Value.Kind() {
		case constant.Bool:
			return NewBoolConstantExpr(constant.BoolVal(value.Value))
		case constant.Int:
			v64, isExact := constant.Uint64Val(value.Value)
			assert(isExact, "inexact constant int")
			return NewConstantExpr(v64, s.executor.Sizeof(value.Type().Underlying()))
		case constant.String:
			str := constant.StringVal(value.Value)
			array := NewArray(0, uint(len(str)))
			for i := 0; i < len(str); i++ {
				array.storeByte(NewConstantExpr64(uint64(i)), NewConstantExpr(uint64(str[i]), 8))
			}
			return array
		case constant.Float:
			panic("glee.Executor: floating point constants are not supported")
		case constant.Complex:
			panic("glee.Executor: complex constants are not supported")
		default:
			panic(fmt.Sprintf("unexpected const: %T", value.Value))
		}
	case *ssa.Function:
		return NewConstantExpr(uint64(uintptr(unsafe.Pointer(value))), s.executor.PointerWidth())
	default:
		if f := s.Frame(); f != nil {
			return f.bindings[value]
		}
		return nil
	}
}

// MustEvalAsExpr is the same as Eval() except that it returns an Expr type.
// Panic if binding is Array or Tuple.
func (s *ExecutionState) MustEvalAsExpr(value ssa.Value) Expr {
	binding := s.Eval(value)
	if binding == nil {
		return nil
	} else if expr, ok := binding.(Expr); ok {
		return expr
	}
	panic(fmt.Sprintf("glee: binding must be an Expr: %T", binding))
}

// EvalAsConstantExpr is the same as Eval() except that it returns an ConstantExpr type.
func (s *ExecutionState) EvalAsConstantExpr(value ssa.Value) (*ConstantExpr, bool) {
	if binding := s.Eval(value); binding == nil {
		return nil, true
	} else if expr, ok := binding.(*ConstantExpr); ok {
		return expr, true
	}
	return nil, false
}

// ExtractCall returns the underlying function reference and arg bindings.
func (s *ExecutionState) ExtractCall(instr ssa.CallInstruction) (fn *ssa.Function, args []Binding) {
	common := instr.Common()

	if _, ok := common.Value.(*ssa.Builtin); !ok {
		// Handle invocation differently if it's a call mode or interface method invocation.
		if common.IsInvoke() {
			// Extract concrete type & pointer.
			iface := s.Eval(common.Value).(*Array)
			typeID := int(s.selectIntAt(iface, 0).(*ConstantExpr).Value)
			typ := s.executor.typesByID[typeID]
			if typ == nil {
				panic(fmt.Sprintf("glee.Executor: type not found: id=%d", typeID))
			}
			data := s.selectIntAt(iface, 1)

			fn = s.executor.prog.LookupMethod(typ, common.Method.Pkg(), common.Method.Name())
			args = append(args, data) // add receiver
		} else {
			addr, ok := s.EvalAsConstantExpr(common.Value)
			if !ok {
				panic(fmt.Sprintf("glee.ExecutionState: expected constant function address"))
			}
			fn = (*ssa.Function)(unsafe.Pointer(uintptr(addr.Value)))
		}
	}

	// Append expressions for arg value.
	for _, arg := range common.Args {
		args = append(args, s.Eval(arg))
	}
	return fn, args
}

// Push adds a frame to the top of the stack.
func (s *ExecutionState) Push(fn *ssa.Function) {
	f := NewStackFrame(s.Frame(), fn)

	f.locals = make([]*Array, len(fn.Locals))
	for i, instr := range fn.Locals {
		width := s.executor.Sizeof(deref(instr.Type()))
		addr, array := s.Alloc(width / 8)
		array.zero()

		f.locals[i] = array
		f.bind(instr, addr)
	}

	s.stack = append(s.stack, f)
}

// Pop returns the current frame from the stack and unbinds the stack variables.
func (s *ExecutionState) Pop() {
	f := s.Frame()
	for _, array := range f.locals {
		s.heap = s.heap.Delete(array.ID)
	}
	s.stack[len(s.stack)-1] = nil
	s.stack = s.stack[:len(s.stack)-1]

	// Mark as finished if no more frames exist.
	if len(s.stack) == 0 {
		s.status = ExecutionStatusFinished
	}
}

// Fork returns a child copy of the given state with the additional constraint.
func (s *ExecutionState) Fork(constraint Expr) *ExecutionState {
	child := s.Clone()
	child.parent = s
	child.covered = make(map[string]map[uint]struct{})
	if constraint != nil {
		child.AddConstraint(constraint)
	}
	s.children = append(s.children, child)
	return child
}

// Done returns true if state encounters a finishing instruction (return, if, etc).
func (s *ExecutionState) Done() bool {
	if s.Terminated() || s.Forked() {
		return true
	}

	instr := s.Instr()
	if instr == nil {
		return false
	}
	switch instr.(type) {
	case *ssa.If, *ssa.Return:
		return true
	default:
		return false
	}
}

// Forked returns true if state has a child state.
func (s *ExecutionState) Forked() bool {
	return len(s.children) > 0
}

// Values computes initial values for all symbolic expressions.
func (s *ExecutionState) Values() ([]*Array, [][]byte, error) {
	arrays := FindArrays(s.constraints...)

	satisfiable, values, err := s.executor.Solver.Solve(s.constraints, arrays)
	if err != nil {
		return nil, nil, err
	} else if !satisfiable {
		return nil, nil, errors.New("unsatisfiable")
	}
	return arrays, values, nil
}

// AddConstraint adds a constraint to the state. Panic if expr is a constant false.
func (s *ExecutionState) AddConstraint(expr Expr) {
	if expr, ok := expr.(*ConstantExpr); ok {
		assert(expr.IsTrue(), "invalid false constraint")
	}

	// Split logical conjunctions into two separate constraints.
	if expr, ok := expr.(*BinaryExpr); ok && expr.Op == AND {
		s.AddConstraint(expr.LHS)
		s.AddConstraint(expr.RHS)
		return
	}

	s.constraints = append(s.constraints, expr)
}

// AddConstraint adds expr to constraints and returns the new constraint list.
// If expr is a binary AND expression then its LHS & RHS are split into
// independent constraints.
func AddConstraint(a []Expr, expr Expr) []Expr {
	if expr, ok := expr.(*BinaryExpr); ok && expr.Op == AND {
		a = AddConstraint(a, expr.LHS)
		a = AddConstraint(a, expr.RHS)
		return a
	}
	return append(a, expr)
}

// Alloc a new array on the heap.
func (s *ExecutionState) Alloc(width uint) (*ConstantExpr, *Array) {
	addr := s.nextAddr()
	array := NewArray(addr, width)
	s.heap = s.heap.Set(addr, array)
	return NewConstantExpr(addr, s.executor.PointerWidth()), array
}

// nextAddr returns the next available address on the heap.
// Ensures the address is always non-zero.
func (s *ExecutionState) nextAddr() uint64 {
	itr := s.heap.Iterator()
	itr.Last()
	if k, v := itr.Prev(); k != nil {
		return k.(uint64) + uint64(v.(*Array).Size)
	}
	return uint64(s.executor.PointerWidth())
}

func (s *ExecutionState) findAllocByAddr(addr *ConstantExpr) *Array {
	if value, _ := s.heap.Get(addr.Value); value != nil {
		return value.(*Array)
	}
	return nil
}

func (s *ExecutionState) findAllocContainingAddr(addr *ConstantExpr) (base *ConstantExpr, array *Array) {
	// Seek to the given address or the next available address.
	itr := s.heap.Iterator()
	if itr.Seek(addr.Value); itr.Done() {
		itr.Last()
	}

	// Move backwards until address range too low.
	for !itr.Done() {
		k, v := itr.Prev()
		key, value := k.(uint64), v.(*Array)

		if addr.Value >= key && addr.Value < key+uint64(value.Size) {
			return NewConstantExpr(key, s.executor.PointerWidth()), value
		} else if addr.Value > key+uint64(value.Size) {
			break // target address above allocation, exit
		}
	}
	return nil, nil
}

// Copy copies the bytes in the value array to the given address.
func (s *ExecutionState) Copy(addr *ConstantExpr, value *Array) {
	base, array := s.findAllocContainingAddr(addr)
	assert(array != nil, "copy: allocation not found: addr=%d", addr.Value)

	newArray := array.Clone()
	for i := uint64(0); i < uint64(value.Size); i++ {
		index := newAddExpr(newSubExpr(addr, base), NewConstantExpr64(i))
		newArray.storeByte(index, value.selectByte(NewConstantExpr64(i)))
	}
	s.heap = s.heap.Set(base.Value, newArray)
}

// Store updates the bytes at addr with value.
// Returns the new copy of the address space.
func (s *ExecutionState) Store(addr *ConstantExpr, value Expr) {
	base, array := s.findAllocContainingAddr(addr)
	assert(array != nil, "store: allocation not found: addr=%d", addr.Value)
	newArray := array.Store(newSubExpr(addr, base), value, s.executor.IsLittleEndian())
	s.heap = s.heap.Set(base.Value, newArray)
}

// selectIntAt returns the i-th pointer-width expression selected from an array.
func (s *ExecutionState) selectIntAt(array *Array, i int) Expr {
	pointerWidth := s.executor.PointerWidth()
	return array.Select(NewConstantExpr32(uint64(i)*uint64(pointerWidth/8)), pointerWidth, s.executor.IsLittleEndian())
}

// storeIntAt returns a new array with the i-th pointer-width element updated.
func (s *ExecutionState) storeIntAt(array *Array, i int, value Expr) *Array {
	pointerWidth := uint64(s.executor.PointerWidth())
	return array.Store(NewConstantExpr64(uint64(i)*(pointerWidth/8)), value, s.executor.IsLittleEndian())
}

// Dump returns the contents of the state and frames as a string.
func (s *ExecutionState) Dump() string {
	var buf bytes.Buffer

	fmt.Fprintln(&buf, "EXECUTION STATE")
	fmt.Fprintln(&buf, "===============")
	fmt.Fprintf(&buf, "status=%s\n", s.status)
	fmt.Fprintf(&buf, "reason=%s\n", s.reason)
	fmt.Fprintln(&buf, "")
	for i := len(s.stack) - 1; i >= 0; i-- {
		fmt.Fprintf(&buf, "== FRAME #%d\n", i)
		fmt.Fprintln(&buf, s.stack[i].Dump())
	}
	fmt.Fprintln(&buf, "")

	fmt.Fprintln(&buf, "== HEAP")
	fmt.Fprintln(&buf, s.dumpHeap())
	fmt.Fprintln(&buf, "")

	fmt.Fprintln(&buf, "== CONSTRAINTS")
	for i, expr := range s.constraints {
		fmt.Fprintf(&buf, "%d. %s\n", i, expr.String())
	}
	return buf.String()
}

func (s *ExecutionState) dumpHeap() string {
	var buf bytes.Buffer
	itr := s.heap.Iterator()
	for {
		k, v := itr.Next()
		if k == nil {
			return buf.String()
		}
		array := v.(*Array)
		fmt.Fprintf(&buf, "%08d %s\n", k.(uint64), array.String())
		for upd := array.Updates; upd != nil; upd = upd.Next {
			fmt.Fprintf(&buf, "  + UPD: I=%s; V=%s\n", upd.Index.String(), upd.Value.String())
		}
		fmt.Fprintln(&buf, "")
	}
}

// ExecutionStatus represents the current status of the execution state.
// The state will also include a reason if the status is not running.
type ExecutionStatus string

const (
	ExecutionStatusRunning  = ExecutionStatus("running")  // has future states
	ExecutionStatusFinished = ExecutionStatus("finished") // clean completion
	ExecutionStatusPanicked = ExecutionStatus("panicked") // panic occurred
	ExecutionStatusFailed   = ExecutionStatus("failed")   // test failed
	ExecutionStatusExited   = ExecutionStatus("exited")   // process exited
)

// StackFrame represents the state of a call into a function.
type StackFrame struct {
	fn       *ssa.Function
	caller   *StackFrame
	locals   []*Array
	bindings map[ssa.Value]Binding

	block *ssa.BasicBlock
	prev  *ssa.BasicBlock
	pc    int
}

// NewStackFrame returns a new instance of StackFrame for a given function.
func NewStackFrame(caller *StackFrame, fn *ssa.Function) *StackFrame {
	return &StackFrame{
		fn:       fn,
		caller:   caller,
		bindings: make(map[ssa.Value]Binding),
		block:    fn.Blocks[0],
		pc:       -1,
	}
}

// Instr returns the current instruction.
func (f *StackFrame) Instr() ssa.Instruction {
	if f.block == nil || f.pc < 0 || f.pc >= len(f.block.Instrs) {
		return nil
	}
	return f.block.Instrs[f.pc]
}

// NextInstr moves the current execution to the next instruction.
func (f *StackFrame) NextInstr() {
	if f.block != nil && f.pc < len(f.block.Instrs) {
		f.pc++
	}
}

// jump moves to dst from the current block.
func (f *StackFrame) jump(dst *ssa.BasicBlock) {
	f.prev, f.block, f.pc = f.block, dst, -1
}

// bind assigns the expression or slice of expressions to a given SSA value.
func (f *StackFrame) bind(value ssa.Value, b Binding) {
	f.bindings[value] = b
}

// Clone returns a copy of the stack frame.
func (f *StackFrame) Clone() *StackFrame {
	other := *f

	other.bindings = make(map[ssa.Value]Binding, len(f.bindings))
	for k := range f.bindings {
		other.bindings[k] = f.bindings[k]
	}

	other.locals = make([]*Array, len(f.locals))
	copy(other.locals, f.locals)

	return &other
}

// BoundValues returns all bound values, sorted by name.
func (f *StackFrame) BoundValues() []ssa.Value {
	a := make([]ssa.Value, 0, len(f.bindings))
	for value := range f.bindings {
		a = append(a, value)
	}

	sort.Slice(a, func(i, j int) bool {
		x, _ := strconv.Atoi(strings.TrimPrefix(a[i].Name(), "t"))
		y, _ := strconv.Atoi(strings.TrimPrefix(a[j].Name(), "t"))
		return x < y
	})

	return a
}

// Dump returns the contents of the frame as a string.
func (f *StackFrame) Dump() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "fn=%s\n", f.fn.String())
	for _, value := range f.BoundValues() {
		binding := f.bindings[value]
		fmt.Fprintf(&buf, "%s (%s)\n%s\n\n", value.Name(), value.Type().String(), binding)
	}
	return buf.String()
}

// Binding represents an object that can be bound to an SSA value.
// This can be either an Expr or a Tuple.
type Binding interface {
	binding()
	String() string
}

func (*BinaryExpr) binding()       {}
func (*CastExpr) binding()         {}
func (*ConcatExpr) binding()       {}
func (*ConstantExpr) binding()     {}
func (*ExtractExpr) binding()      {}
func (*NotExpr) binding()          {}
func (*NotOptimizedExpr) binding() {}
func (*SelectExpr) binding()       {}
func (*Array) binding()            {}
func (Tuple) binding()             {}

// uint64Comparer compares two 64-bit unsigned integers. Implements immutable.Comparer.
type uint64Comparer struct{}

// Compare returns -1 if a is less than b, returns 1 if a is greater than b, and
// returns 0 if a is equal to b. Panic if a or b is not an int.
func (c *uint64Comparer) Compare(a, b interface{}) int {
	if i, j := a.(uint64), b.(uint64); i < j {
		return -1
	} else if i > j {
		return 1
	}
	return 0
}
