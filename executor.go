package glee

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"log"
	"math/rand"
	"path/filepath"
	"runtime"
	"sort"

	"golang.org/x/tools/go/ssa"
)

var (
	ErrNoStateAvailable       = errors.New("glee: no state available")
	ErrNoInstructionAvailable = errors.New("glee: no instruction available")
)

type Executor struct {
	fn         *ssa.Function                // entry function
	root       *ExecutionState              // initial state
	states     map[*ExecutionState]struct{} // all states
	globals    map[*ssa.Global]Expr         // global variables
	stateIDSeq int                          // autoincrementing state ID

	prog *ssa.Program                // entire program, ease-of-use var
	fns  map[funcKey]FunctionHandler // registered function handlers

	// Mapping of types to generated IDs and back.
	// This is used for deterministically assigning pointer values.
	typeIDs   map[types.Type]int
	typesByID map[int]types.Type

	// OS & architecture settings for the executor.
	// See `go tool dist list` for a list of valid combinations.
	OS   string
	Arch string

	// Used for solving symbolic values.
	// Must set before execution.
	Solver Solver

	// Search strategy for the executor. Defaults to depth-first.
	Searcher Searcher
}

// NewExecutor returns a new instance of Executor.
func NewExecutor(fn *ssa.Function) *Executor {
	e := &Executor{
		fn:      fn,
		globals: make(map[*ssa.Global]Expr),

		prog: fn.Prog,
		fns:  make(map[funcKey]FunctionHandler),

		typeIDs:   make(map[types.Type]int),
		typesByID: make(map[int]types.Type),

		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Searcher: NewDFSSearcher(),
	}

	// Register all program types in deterministic order.
	for _, typ := range programTypes(fn.Prog) {
		typeID := len(e.typeIDs) + 1
		e.typeIDs[typ] = typeID
		e.typesByID[typeID] = typ
	}

	// Default registrations.
	pkgName := "github.com/benbjohnson/glee"
	e.Register(pkgName, "Assert", execAssert)
	e.Register(pkgName, "Byte", execInt)
	e.Register(pkgName, "Int", execInt)
	e.Register(pkgName, "Int8", execInt)
	e.Register(pkgName, "Int16", execInt)
	e.Register(pkgName, "Int32", execInt)
	e.Register(pkgName, "Int64", execInt)
	e.Register(pkgName, "Uint", execInt)
	e.Register(pkgName, "Uint8", execInt)
	e.Register(pkgName, "Uint16", execInt)
	e.Register(pkgName, "Uint32", execInt)
	e.Register(pkgName, "Uint64", execInt)
	e.Register(pkgName, "ByteSlice", execByteSlice)
	e.Register(pkgName, "String", execString)
	e.Register("", "copy", execCopy)
	e.Register("", "len", execLen)
	e.Register("testing", "Fatal", execTestingFatal)

	// Initialize entry state.
	e.root = NewExecutionState(e, fn)
	e.root.id = e.nextStateID()

	// Add state to searcher.
	e.states = map[*ExecutionState]struct{}{e.root: struct{}{}}
	e.Searcher.AddState(e.root)

	return e
}

// RootState returns the initial state for the function execution.
func (e *Executor) RootState() *ExecutionState { return e.root }

// nextStateID returns the next autoincrementing state ID.
func (e *Executor) nextStateID() int {
	e.stateIDSeq++
	return e.stateIDSeq
}

// Register registers a function handler for a given function.
// Every invocation of the given function will be delegated to the handler.
func (e *Executor) Register(path, name string, h FunctionHandler) {
	e.fns[funcKey{path, name}] = h
}

// ExecuteNextState executes the next available state. This can be called
// continually until ErrNoStateAvailable is returned.
func (e *Executor) ExecuteNextState() (*ExecutionState, error) {
	if !isValidOSArch(e.OS, e.Arch) {
		return nil, errors.New("invalid os/arch combination")
	}

	state := e.Searcher.SelectState()
	if state == nil {
		return nil, ErrNoStateAvailable
	}

	log.Printf("[state] begin: %s", state.Position().String())
	defer log.Printf("")

	// Loop until new states available or completion.
	for {
		if err := e.executeNextInstruction(state); err == ErrNoInstructionAvailable {
			break
		} else if err != nil {
			return state, err
		} else if state.Done() {
			break
		}
	}
	return state, nil
}

func (e *Executor) executeNextInstruction(state *ExecutionState) (err error) {
	// Find the next available instruction on the current frame or pop
	// up to the caller if no more instructions remain. If no more frames
	// exist then execution is done.
	var frame *StackFrame
	for {
		frame = state.Frame()
		if frame == nil {
			return ErrNoInstructionAvailable
		}

		// Continue if instruction exists.
		state.Frame().NextInstr()
		if state.Frame().Instr() != nil {
			break
		}
		state.Pop()
	}

	// Log each non-debug line of execution.
	instr := state.Instr()
	if _, ok := instr.(*ssa.DebugRef); !ok {
		pos := state.Position()
		pos.Filename = filepath.Base(pos.Filename)
		pos.Column = 0
		log.Printf("[exec] %s: %s (%T)", pos, instr.String(), instr)
	}

	switch instr := instr.(type) {
	case *ssa.Alloc:
		return e.executeAllocInstr(state, instr)
	case *ssa.BinOp:
		return e.executeBinOpInstr(state, instr)
	case *ssa.Call:
		return e.executeCallInstr(state, instr)
	case *ssa.ChangeInterface:
		return e.executeChangeInterfaceInstr(state, instr)
	case *ssa.ChangeType:
		return e.executeChangeTypeInstr(state, instr)
	case *ssa.Convert:
		return e.executeConvertInstr(state, instr)
	case *ssa.DebugRef:
		return nil // nop
	case *ssa.Defer:
		return e.executeDeferInstr(state, instr)
	case *ssa.Extract:
		return e.executeExtractInstr(state, instr)
	case *ssa.Field:
		return e.executeFieldInstr(state, instr)
	case *ssa.FieldAddr:
		return e.executeFieldAddrInstr(state, instr)
	case *ssa.Go:
		return errors.New("goroutines are not currently supported")
	case *ssa.If:
		return e.executeIfInstr(state, instr)
	case *ssa.Index:
		return e.executeIndexInstr(state, instr)
	case *ssa.IndexAddr:
		return e.executeIndexAddrInstr(state, instr)
	case *ssa.Jump:
		return e.executeJumpInstr(state, instr)
	case *ssa.Lookup:
		return e.executeLookupInstr(state, instr)
	case *ssa.MakeChan:
		return e.executeMakeChanInstr(state, instr)
	case *ssa.MakeClosure:
		return e.executeMakeClosureInstr(state, instr)
	case *ssa.MakeInterface:
		return e.executeMakeInterfaceInstr(state, instr)
	case *ssa.MakeMap:
		return e.executeMakeMapInstr(state, instr)
	case *ssa.MakeSlice:
		return e.executeMakeSliceInstr(state, instr)
	case *ssa.MapUpdate:
		return e.executeMapUpdateInstr(state, instr)
	case *ssa.Next:
		return e.executeNextInstr(state, instr)
	case *ssa.Panic:
		return e.executePanicInstr(state, instr)
	case *ssa.Phi:
		return e.executePhiInstr(state, instr)
	case *ssa.Range:
		return e.executeRangeInstr(state, instr)
	case *ssa.Return:
		return e.executeReturnInstr(state, instr)
	case *ssa.RunDefers:
		return e.executeRunDefersInstr(state, instr)
	case *ssa.Select:
		return e.executeSelectInstr(state, instr)
	case *ssa.Send:
		return e.executeSendInstr(state, instr)
	case *ssa.Slice:
		return e.executeSliceInstr(state, instr)
	case *ssa.Store:
		return e.executeStoreInstr(state, instr)
	case *ssa.TypeAssert:
		return e.executeTypeAssertInstr(state, instr)
	case *ssa.UnOp:
		return e.executeUnOpInstr(state, instr)
	default:
		return errors.New("illegal instruction")
	}
}

func (e *Executor) executeAllocInstr(state *ExecutionState, instr *ssa.Alloc) error {
	// Non-heap allocs are allocated when pushing function onto stack.
	if !instr.Heap {
		return nil
	}

	// Allocate zero-initialized and bind address to instruction.
	size := e.Sizeof(deref(instr.Type())) / 8
	addr, array := state.Alloc(size)
	array.zero()
	state.Frame().bind(instr, addr)

	log.Printf("[alloc] type=%s addr=%d size=%d", instr.Type(), addr.Value, size)

	return nil
}

func (e *Executor) executeBinOpInstr(state *ExecutionState, instr *ssa.BinOp) error {
	switch typ := instr.X.Type().Underlying().(type) {
	case *types.Interface:
		return e.executeBinOpInstrInterface(state, instr)
	case *types.Basic:
		info := typ.Info()
		if info&types.IsBoolean != 0 {
			return e.executeBinOpInstrBoolean(state, instr)
		} else if info&types.IsInteger != 0 {
			return e.executeBinOpInstrInteger(state, instr, types.IsUnsigned == 0)
		} else if info&types.IsFloat != 0 {
			return e.executeBinOpInstrFloat(state, instr)
		} else if info&types.IsComplex != 0 {
			return e.executeBinOpInstrComplex(state, instr)
		} else if info&types.IsString != 0 {
			return e.executeBinOpInstrString(state, instr)
		}
		return errors.New("unexpected binop basic type")
	default:
		return fmt.Errorf("unexpected binop X type: %T", typ)
	}
}

func (e *Executor) executeBinOpInstrInterface(state *ExecutionState, instr *ssa.BinOp) error {
	x, y := state.Eval(instr.X).(*Array), state.Eval(instr.Y).(*Array)
	switch instr.Op {
	case token.EQL:
		state.Frame().bind(instr, x.Equal(y))
		return nil
	case token.NEQ:
		state.Frame().bind(instr, x.NotEqual(y))
		return nil
	default:
		return errors.New("invalid boolean binop operator")
	}
}

func (e *Executor) executeBinOpInstrBoolean(state *ExecutionState, instr *ssa.BinOp) error {
	x, y := state.Eval(instr.X).(Expr), state.Eval(instr.Y).(Expr)
	switch instr.Op {
	case token.AND:
		state.Frame().bind(instr, NewBinaryExpr(AND, x, y))
		return nil
	case token.OR:
		state.Frame().bind(instr, NewBinaryExpr(OR, x, y))
		return nil
	default:
		return errors.New("invalid boolean binop operator")
	}
}

func (e *Executor) executeBinOpInstrInteger(state *ExecutionState, instr *ssa.BinOp, signed bool) error {
	x, y := state.Eval(instr.X).(Expr), state.Eval(instr.Y).(Expr)

	switch instr.Op {
	case token.ADD:
		state.Frame().bind(instr, NewBinaryExpr(ADD, x, y))
		return nil
	case token.SUB:
		state.Frame().bind(instr, NewBinaryExpr(SUB, x, y))
		return nil
	case token.MUL:
		state.Frame().bind(instr, NewBinaryExpr(MUL, x, y))
		return nil
	case token.QUO:
		if signed {
			state.Frame().bind(instr, NewBinaryExpr(SDIV, x, y))
		} else {
			state.Frame().bind(instr, NewBinaryExpr(UDIV, x, y))
		}
		return nil
	case token.REM: // unsigned vs signed
		if signed {
			state.Frame().bind(instr, NewBinaryExpr(SREM, x, y))
		} else {
			state.Frame().bind(instr, NewBinaryExpr(UREM, x, y))
		}
		return nil
	case token.AND:
		state.Frame().bind(instr, NewBinaryExpr(AND, x, y))
		return nil
	case token.OR:
		state.Frame().bind(instr, NewBinaryExpr(OR, x, y))
		return nil
	case token.XOR:
		state.Frame().bind(instr, NewBinaryExpr(XOR, x, y))
		return nil
	case token.SHL:
		state.Frame().bind(instr, NewBinaryExpr(SHL, x, y))
		return nil
	case token.SHR:
		if signed {
			state.Frame().bind(instr, NewBinaryExpr(ASHR, x, y))
		} else {
			state.Frame().bind(instr, NewBinaryExpr(LSHR, x, y))
		}
		return nil
	case token.AND_NOT:
		state.Frame().bind(instr, NewBinaryExpr(XOR, x, y))
		return nil
	case token.EQL:
		state.Frame().bind(instr, NewBinaryExpr(EQ, x, y))
		return nil
	case token.NEQ:
		state.Frame().bind(instr, NewBinaryExpr(NE, x, y))
		return nil
	case token.LSS:
		if signed {
			state.Frame().bind(instr, NewBinaryExpr(SLT, x, y))
		} else {
			state.Frame().bind(instr, NewBinaryExpr(ULT, x, y))
		}
		return nil
	case token.LEQ:
		if signed {
			state.Frame().bind(instr, NewBinaryExpr(SLE, x, y))
		} else {
			state.Frame().bind(instr, NewBinaryExpr(ULE, x, y))
		}
		return nil
	case token.GTR:
		if signed {
			state.Frame().bind(instr, NewBinaryExpr(SGT, x, y))
		} else {
			state.Frame().bind(instr, NewBinaryExpr(UGT, x, y))
		}
		return nil
	case token.GEQ:
		if signed {
			state.Frame().bind(instr, NewBinaryExpr(SGE, x, y))
		} else {
			state.Frame().bind(instr, NewBinaryExpr(UGE, x, y))
		}
		return nil
	default:
		return errors.New("invalid integer binop operator")
	}
}

func (e *Executor) executeBinOpInstrFloat(state *ExecutionState, instr *ssa.BinOp) error {
	return errors.New("floating-point operations are not supported")
}

func (e *Executor) executeBinOpInstrComplex(state *ExecutionState, instr *ssa.BinOp) error {
	return errors.New("complex number operations are not supported")
}

func (e *Executor) executeBinOpInstrString(state *ExecutionState, instr *ssa.BinOp) error {
	switch instr.Op {
	case token.ADD:
		return e.executeBinOpInstrStringADD(state, instr)
	case token.EQL:
		x, y := state.Eval(instr.X).(*Array), state.Eval(instr.Y).(*Array)
		state.Frame().bind(instr, x.Equal(y))
		return nil
	case token.NEQ:
		x, y := state.Eval(instr.X).(*Array), state.Eval(instr.Y).(*Array)
		state.Frame().bind(instr, x.NotEqual(y))
		return nil
	case token.LSS, token.LEQ, token.GTR, token.GEQ:
		return e.executeBinOpInstrStringCompare(state, instr)
	default:
		return errors.New("invalid string binop operator")
	}
}

func (e *Executor) executeBinOpInstrStringADD(state *ExecutionState, instr *ssa.BinOp) error {
	x, y := state.Eval(instr.X).(*Array), state.Eval(instr.Y).(*Array)

	log.Printf("[binop] str-add x=%s y=%s", x, y)

	// Return either x or y if the other is zero length.
	if x.Size == 0 {
		state.Frame().bind(instr, y)
		return nil
	} else if y.Size == 0 {
		state.Frame().bind(instr, x)
		return nil
	}

	// If x & y are non-blank then create a new array and copy all bytes.
	array := NewArray(0, x.Size+y.Size)
	for i := uint(0); i < x.Size; i++ {
		index := NewConstantExpr64(uint64(i))
		array.storeByte(index, x.selectByte(index))
	}
	for i := uint(0); i < y.Size; i++ {
		array.storeByte(NewConstantExpr64(uint64(x.Size+i)), y.selectByte(NewConstantExpr64(uint64(i))))
	}

	// Bind new array to instruction.
	state.Frame().bind(instr, array)

	return nil
}

// executeBinOpInstrStringCompare implements LSS, LTE, GTR, & GTE string comparisons.
func (e *Executor) executeBinOpInstrStringCompare(state *ExecutionState, instr *ssa.BinOp) error {
	x := state.Eval(instr.X).(*Array)
	y := state.Eval(instr.Y).(*Array)

	// Empty strings cannot be less than or greater than one another.
	if instr.Op == token.LSS || instr.Op == token.GTR {
		if x.Size == 0 && y.Size == 0 {
			state.Frame().bind(instr, NewBoolConstantExpr(false))
			return nil
		}
	}

	// Use the lower size.
	n := uint64(x.Size)
	if n > uint64(y.Size) {
		n = uint64(y.Size)
	}

	// Generate all selection expressions once to conserve memory.
	xSelectExprs, ySelectExprs := make([]Expr, n), make([]Expr, n)
	for i := uint64(0); i < n; i++ {
		index := NewConstantExpr64(i)
		xSelectExprs[i] = x.selectByte(index)
		ySelectExprs[i] = y.selectByte(index)
	}

	// Generate OR-concatenated expression for every byte.
	var cond Expr
	for i := uint64(0); i < n; i++ {
		// Check the current byte for given operation.
		// Last LSS/LEQ byte can be equal iif x is shorter or if equal len (LEQ only).
		// Last GTR/GEQ byte can be equal iif x is longer or if equal len (GEQ only).
		var base Expr
		switch instr.Op {
		case token.LSS, token.LEQ:
			if i == n-1 && (x.Size < y.Size || (x.Size == y.Size && instr.Op == token.LEQ)) {
				base = newUleExpr(xSelectExprs[i], ySelectExprs[i]) // last byte, short x or equal len (LEQ)
			} else {
				base = newUltExpr(xSelectExprs[i], ySelectExprs[i])
			}
		case token.GTR, token.GEQ:
			if i == n-1 && (x.Size > y.Size || (x.Size == y.Size && instr.Op == token.GEQ)) {
				base = newUleExpr(ySelectExprs[i], xSelectExprs[i]) // reverse
			} else {
				base = newUltExpr(ySelectExprs[i], xSelectExprs[i]) // reverse
			}
		}

		// Ensure all previous bytes are equal.
		for j := uint64(0); j < i; j++ {
			base = newAndExpr(base, newEqExpr(xSelectExprs[j], ySelectExprs[j]))
		}

		// OR-concat to the current expression.
		if i == 0 {
			cond = base
		} else {
			cond = newOrExpr(cond, base)
		}
	}

	// Bind condition expression to instruction.
	state.Frame().bind(instr, cond)
	return nil
}

func (e *Executor) executeBinOpInstrStringLEQ(state *ExecutionState, instr *ssa.BinOp) error {
	return fmt.Errorf("glee.Executor: string comparison is not supported")
}

func (e *Executor) executeBinOpInstrStringGTR(state *ExecutionState, instr *ssa.BinOp) error {
	return fmt.Errorf("glee.Executor: string comparison is not supported")
}

func (e *Executor) executeBinOpInstrStringGEQ(state *ExecutionState, instr *ssa.BinOp) error {
	return fmt.Errorf("glee.Executor: string comparison is not supported")
}

func (e *Executor) executeCallInstr(state *ExecutionState, instr *ssa.Call) error {
	// Handle builtin functions separately.
	if builtin, ok := instr.Call.Value.(*ssa.Builtin); ok {
		registered := e.fns[funcKey{"", builtin.Name()}]
		if registered == nil {
			panic(fmt.Sprintf("glee.Executor: unregistered builtin function: %s", builtin.Name()))
		}
		return registered(state, instr)
	}

	// Lookup if function is registered with executor and defer execution.
	fn, args := state.ExtractCall(instr)
	path, name := fn.Pkg.Pkg.Path(), fn.Name()
	if registered, ok := e.fns[funcKey{path, name}]; ok {
		return registered(state, instr)
	}

	// Move execution to the new frame & bind arguments.
	log.Printf("[fork] call: %s %s", path, name)
	newState := state.Fork(nil)
	newState.id = e.nextStateID()
	newState.Push(fn)
	for i, arg := range args {
		newState.Frame().bind(fn.Params[i], arg)
	}
	e.Searcher.AddState(newState)

	return nil
}

func (e *Executor) executeChangeInterfaceInstr(state *ExecutionState, instr *ssa.ChangeInterface) error {
	state.Frame().bind(instr, state.Eval(instr.X))
	return nil
}

func (e *Executor) executeChangeTypeInstr(state *ExecutionState, instr *ssa.ChangeType) error {
	x := state.Eval(instr.X)
	state.Frame().bind(instr, x)
	return nil
}

func (e *Executor) executeConvertInstr(state *ExecutionState, instr *ssa.Convert) error {
	srcType, dstType := instr.X.Type().Underlying(), instr.Type().Underlying()

	switch srcType := srcType.(type) {
	case *types.Pointer:
		if dstType, ok := dstType.(*types.Basic); !ok || dstType.Kind() != types.UnsafePointer {
			return fmt.Errorf("glee.Executor: unsupported pointer conversion")
		}
		state.Frame().bind(instr, state.MustEvalAsExpr(instr.X))
		return nil

	case *types.Slice:
		switch srcType.Elem().(*types.Basic).Kind() {
		case types.Byte:
			return e.executeConvertInstrByteSliceToString(state, instr)
		case types.Rune:
			return fmt.Errorf("glee.Executor: rune-to-string conversion is not supported")
		default:
			return fmt.Errorf("glee.Executor: unsupported slice conversion: %s", srcType.Elem())
		}

	case *types.Basic:
		if srcType.Info()&types.IsInteger != 0 {
			if dstType, ok := dstType.(*types.Basic); ok && dstType.Kind() == types.String {
				return fmt.Errorf("glee.Executor: int-to-string conversion is not supported")
			}
		}

		if srcType.Kind() == types.String {
			switch dstType := dstType.(type) {
			case *types.Slice:
				switch dstType.Elem().(*types.Basic).Kind() {
				case types.Rune:
					return fmt.Errorf("glee.Executor: string-to-rune conversion is not supported")
				case types.Byte:
					return e.executeConvertInstrStringToByteSlice(state, instr)
				}
			case *types.Basic:
				if dstType.Kind() == types.String {
					state.Frame().bind(instr, state.Eval(instr.X)) // nop
					return nil
				}
			}
			return fmt.Errorf("glee.Executor: unsupported string conversion: %s", dstType)
		}

		if srcType.Kind() == types.UnsafePointer {
			return fmt.Errorf("glee.Executor: unsafe.Pointer conversion is not supported")
		}

		if srcType.Info()&types.IsComplex != 0 {
			return fmt.Errorf("glee.Executor: complex type conversion is not supported")
		} else if srcType.Info()&types.IsFloat != 0 {
			return fmt.Errorf("glee.Executor: floating point type conversion is not supported")
		} else if (srcType.Info()&types.IsInteger == 0) && (srcType.Info()&types.IsUnsigned == 0) {
			return fmt.Errorf("glee.Executor: unsupported basic type conversion: %s", srcType)
		}

		value := state.MustEvalAsExpr(instr.X)
		signed := srcType.Info()&types.IsUnsigned == 0
		state.Frame().bind(instr, NewCastExpr(value, e.Sizeof(dstType), signed))
		return nil

	default:
		return fmt.Errorf("glee.Executor: unsupported type conversion: %s", srcType)
	}
}

func (e *Executor) executeConvertInstrByteSliceToString(state *ExecutionState, instr *ssa.Convert) error {
	hdr := state.Eval(instr.X).(*Array)

	log.Printf("[convert] []byte-to-string: %s", hdr)

	// Find data using slice header pointer. Must be a constant expression.
	ptr, ok := state.selectIntAt(hdr, 0).(*ConstantExpr)
	if !ok {
		return fmt.Errorf("glee.Executor: cannot read non-constant SliceHeader.Data field")
	}

	// Find length of slice.
	length, ok := state.selectIntAt(hdr, 1).(*ConstantExpr)
	if !ok {
		return fmt.Errorf("glee.Executor: cannot read non-constant SliceHeader.Len field")
	}

	// Find the array at the given address.
	base, src := state.findAllocContainingAddr(ptr)
	if src == nil {
		return fmt.Errorf("glee.Executor: byte slice data allocation not found: %d", ptr.Value)
	}
	offset := ptr.Value - base.Value

	// Copy values from byte slice data to new array.
	dst := NewArray(0, uint(length.Value))
	for i := uint64(0); i < length.Value; i++ {
		dst.storeByte(NewConstantExpr64(i), src.selectByte(NewConstantExpr64(offset+i)))
	}

	// Bind new array to instruction.
	state.Frame().bind(instr, dst)
	return nil
}

func (e *Executor) executeConvertInstrStringToByteSlice(state *ExecutionState, instr *ssa.Convert) error {
	x := state.Eval(instr.X).(*Array)
	length := NewConstantExpr(uint64(x.Size), e.PointerWidth())

	// Build underlying array and copy bytes.
	addr, array := state.Alloc(x.Size)
	for i := uint64(0); i < uint64(x.Size); i++ {
		index := NewConstantExpr64(i)
		array.storeByte(index, x.selectByte(index))
	}

	// Build slice header.
	_, hdr := state.Alloc(e.PointerWidth() * 3)
	hdr = state.storeIntAt(hdr, 0, addr)   // data
	hdr = state.storeIntAt(hdr, 1, length) // len
	hdr = state.storeIntAt(hdr, 2, length) // cap
	state.heap = state.heap.Set(hdr.ID, hdr)

	// Bind header to instruction.
	state.Frame().bind(instr, hdr)

	return nil
}

func (e *Executor) executeDeferInstr(state *ExecutionState, instr *ssa.Defer) error {
	return fmt.Errorf("glee.Executor: defer is not supported")
}

func (e *Executor) executeExtractInstr(state *ExecutionState, instr *ssa.Extract) error {
	tuple := state.Eval(instr.Tuple).(Tuple)
	state.Frame().bind(instr, tuple[instr.Index])
	return nil
}

func (e *Executor) executeFieldInstr(state *ExecutionState, instr *ssa.Field) error {
	return fmt.Errorf("glee.Executor: *ssa.Field instruction not supported")
}

func (e *Executor) executeFieldAddrInstr(state *ExecutionState, instr *ssa.FieldAddr) error {
	// TODO(BBJ): Handle nil instr.X

	// Retrieve type and field layout.
	ptrType := instr.X.Type().Underlying().(*types.Pointer)
	structType := ptrType.Elem().Underlying().(*types.Struct)
	offsets := e.Sizes().Offsetsof(structFields(structType))
	fieldOffset := offsets[instr.Field]

	// Find base address of the structure. Must be a constrant address currently.
	base := state.Eval(instr.X).(*ConstantExpr)

	log.Printf("[field] base=%d offset=%d", base.Value, fieldOffset)

	// Compute offset from base address to field address.
	expr := NewBinaryExpr(ADD, base, NewConstantExpr(uint64(fieldOffset), e.PointerWidth()))
	state.Frame().bind(instr, expr)

	return nil
}

func (e *Executor) executeIndexInstr(state *ExecutionState, instr *ssa.Index) error {
	return fmt.Errorf("glee.Executor: *ssa.Index instruction not supported")
}

func (e *Executor) executeIndexAddrInstr(state *ExecutionState, instr *ssa.IndexAddr) error {
	switch typ := instr.X.Type().(type) {
	case *types.Array:
		return e.executeIndexAddrInstrArray(state, instr, typ)
	case *types.Slice:
		return e.executeIndexAddrInstrSlice(state, instr, typ)
	default:
		return fmt.Errorf("glee.Executor: unexpected IndexAddr.X type: %T", typ)
	}
}

func (e *Executor) executeIndexAddrInstrArray(state *ExecutionState, instr *ssa.IndexAddr, typ *types.Array) error {
	x := state.Eval(instr.X).(*Array)
	index := state.MustEvalAsExpr(instr.Index)

	indexBytes := newMulExpr(index, NewConstantExpr(uint64(e.Sizeof(typ.Elem())/8), e.PointerWidth()))
	state.Frame().bind(instr, newAddExpr(NewConstantExpr(x.ID, e.PointerWidth()), indexBytes))
	return nil
}

func (e *Executor) executeIndexAddrInstrSlice(state *ExecutionState, instr *ssa.IndexAddr, typ *types.Slice) error {
	x := state.Eval(instr.X).(*Array)
	index := state.MustEvalAsExpr(instr.Index)

	indexBytes := newMulExpr(index, NewConstantExpr(uint64(e.Sizeof(typ.Elem())/8), e.PointerWidth()))
	state.Frame().bind(instr, newAddExpr(state.selectIntAt(x, 0), indexBytes))
	return nil
}

func (e *Executor) executeLookupInstr(state *ExecutionState, instr *ssa.Lookup) error {
	switch typ := instr.X.Type().(type) {
	case *types.Basic:
		return e.executeLookupInstrString(state, instr)
	case *types.Map:
		return e.executeLookupInstrMap(state, instr)
	default:
		return fmt.Errorf("glee.Executor: unexpected Lookup.X type: %T", typ)
	}
}

func (e *Executor) executeLookupInstrString(state *ExecutionState, instr *ssa.Lookup) error {
	x := state.Eval(instr.X).(*Array)
	index := newZExtExpr(state.MustEvalAsExpr(instr.Index), 64)

	state.Frame().bind(instr, x.selectByte(index))
	return nil
}

func (e *Executor) executeLookupInstrMap(state *ExecutionState, instr *ssa.Lookup) error {
	return fmt.Errorf("glee.Executor: map lookup is not supported")
}

func (e *Executor) executeMakeChanInstr(state *ExecutionState, instr *ssa.MakeChan) error {
	return fmt.Errorf("glee.Executor: channels are not supported")
}

func (e *Executor) executeMakeClosureInstr(state *ExecutionState, instr *ssa.MakeClosure) error {
	return fmt.Errorf("glee.Executor: closures are not supported")
}

func (e *Executor) executeMakeInterfaceInstr(state *ExecutionState, instr *ssa.MakeInterface) error {
	typeID := uint64(e.typeIDs[instr.X.Type()])

	// Build interface element that contains two pointers.
	// One pointer to the type and one to the data.
	_, iface := state.Alloc((e.PointerWidth() * 2) / 8)
	iface = state.storeIntAt(iface, 0, NewConstantExpr(typeID, e.PointerWidth()))
	iface = state.storeIntAt(iface, 1, state.MustEvalAsExpr(instr.X))
	state.heap = state.heap.Set(iface.ID, iface)

	state.Frame().bind(instr, iface)
	return nil
}

func (e *Executor) executeMakeMapInstr(state *ExecutionState, instr *ssa.MakeMap) error {
	return fmt.Errorf("glee.Executor: map instantiation is not supported")
}

func (e *Executor) executeMakeSliceInstr(state *ExecutionState, instr *ssa.MakeSlice) error {
	typ := instr.Type().(*types.Slice)

	// Evaluate arguments.
	length, ok := state.EvalAsConstantExpr(instr.Len)
	if !ok {
		return fmt.Errorf("glee.Executor: make slice len must be a constant")
	}
	capacity, ok := state.EvalAsConstantExpr(instr.Cap)
	if !ok {
		return fmt.Errorf("glee.Executor: make slice cap must be a constant")
	} else if capacity == nil {
		capacity = length
	}

	// Build underlying array & initialize to zero value.
	elemSizeBytes := (e.Sizeof(typ.Elem()) / 8)
	addr, array := state.Alloc(uint(capacity.Value) * elemSizeBytes)
	array.zero()

	// Build slice header.
	_, hdr := state.Alloc(e.PointerWidth() * 3)
	hdr = state.storeIntAt(hdr, 0, addr)     // data
	hdr = state.storeIntAt(hdr, 1, length)   // len
	hdr = state.storeIntAt(hdr, 2, capacity) // cap

	// Bind header to instruction.
	state.Frame().bind(instr, hdr)

	return nil
}

func (e *Executor) executeMapUpdateInstr(state *ExecutionState, instr *ssa.MapUpdate) error {
	return fmt.Errorf("glee.Executor: map update is not supported")
}

func (e *Executor) executeNextInstr(state *ExecutionState, instr *ssa.Next) error {
	return fmt.Errorf("glee.Executor: range next is not supported")
}

func (e *Executor) executePanicInstr(state *ExecutionState, instr *ssa.Panic) error {
	return fmt.Errorf("glee.Executor: panic is not supported")
}

func (e *Executor) executeRangeInstr(state *ExecutionState, instr *ssa.Range) error {
	return fmt.Errorf("glee.Executor: range is not supported")
}

func (e *Executor) executeRunDefersInstr(state *ExecutionState, instr *ssa.RunDefers) error {
	return fmt.Errorf("glee.Executor: defer is not supported")
}

func (e *Executor) executeSelectInstr(state *ExecutionState, instr *ssa.Select) error {
	return fmt.Errorf("glee.Executor: select is not supported")
}

func (e *Executor) executeSendInstr(state *ExecutionState, instr *ssa.Send) error {
	return fmt.Errorf("glee.Executor: send is not supported")
}

func (e *Executor) executeSliceInstr(state *ExecutionState, instr *ssa.Slice) error {
	switch typ := deref(instr.X.Type()).(type) {
	case *types.Array:
		return e.executeSliceInstrArray(state, instr)
	case *types.Basic:
		return e.executeSliceInstrString(state, instr)
	case *types.Slice:
		return e.executeSliceInstrSlice(state, instr)
	default:
		return fmt.Errorf("glee.Executor.executeSliceInstr(): unexpected slice type: %T", typ)
	}

}

func (e *Executor) executeSliceInstrArray(state *ExecutionState, instr *ssa.Slice) error {
	addr, ok := state.EvalAsConstantExpr(instr.X)
	if !ok {
		return fmt.Errorf("glee.Executor: array slice address must be a constant expression")
	}
	array := state.findAllocByAddr(addr)
	if array == nil {
		return fmt.Errorf("glee.Executor: cannot find array allocation: %d", addr.Value)
	}

	lo := state.MustEvalAsExpr(instr.Low)
	hi := state.MustEvalAsExpr(instr.High)
	max := state.MustEvalAsExpr(instr.Max)

	log.Printf("[slice] array low=%v high=%v max=%v", lo, hi, max)

	// Determine element width.
	pointerWidth := e.PointerWidth()
	typ := instr.Type().(*types.Slice)
	elemWidth := NewConstantExpr(uint64(e.Sizeof(typ.Elem()))/8, pointerWidth)

	// Set index defaults.
	if lo == nil {
		lo = NewConstantExpr(0, pointerWidth)
	}
	if hi == nil {
		hi = NewConstantExpr(uint64(array.Size), pointerWidth)
	}
	if max == nil {
		max = NewConstantExpr(uint64(array.Size), pointerWidth)
	}

	// Copy to new header with updated data/len/cap.
	_, hdr := state.Alloc((pointerWidth / 8) * 3)
	hdr = state.storeIntAt(hdr, 0, newAddExpr(addr, newMulExpr(lo, elemWidth))) // data
	hdr = state.storeIntAt(hdr, 1, newSubExpr(hi, lo))                          // len
	hdr = state.storeIntAt(hdr, 2, newSubExpr(max, lo))                         // cap
	state.heap = state.heap.Set(hdr.ID, hdr)

	// Bind header to instruction.
	state.Frame().bind(instr, hdr)

	return nil
}

func (e *Executor) executeSliceInstrString(state *ExecutionState, instr *ssa.Slice) error {
	x := state.Eval(instr.X).(*Array)

	// Ensure low index is constant.
	lo, ok := state.EvalAsConstantExpr(instr.Low)
	if !ok {
		return fmt.Errorf("glee.Executor: string slice low index must be a constant expression")
	} else if lo == nil {
		lo = NewConstantExpr64(0)
	}

	// Ensure high index is constant.
	hi, ok := state.EvalAsConstantExpr(instr.High)
	if !ok {
		return fmt.Errorf("glee.Executor: string slice high index must be a constant expression")
	} else if hi == nil {
		hi = NewConstantExpr64(uint64(x.Size))
	}

	log.Printf("[slice] string low=%v high=%v", lo, hi)

	// Verify low & high are inbounds.
	if hi.Value > uint64(x.Size) || lo.Value > uint64(x.Size) {
		state.status = ExecutionStatusPanicked
		state.reason = "slice bounds out of range"
		return nil
	}

	// Copy substring to new array.
	array := NewArray(0, uint(hi.Value-lo.Value))
	for i := uint(0); i < array.Size; i++ {
		array.storeByte(NewConstantExpr64(uint64(i)), x.selectByte(NewConstantExpr64(uint64(i)+lo.Value)))
	}

	// Bind substring to instruction.
	state.Frame().bind(instr, array)

	return nil
}

func (e *Executor) executeSliceInstrSlice(state *ExecutionState, instr *ssa.Slice) error {
	x := state.Eval(instr.X).(*Array)
	lo := state.MustEvalAsExpr(instr.Low)
	hi := state.MustEvalAsExpr(instr.High)
	max := state.MustEvalAsExpr(instr.Max)

	log.Printf("[slice] slice low=%v high=%v max=%v, id=#%d", lo, hi, max, x.ID)

	// Determine element width.
	pointerWidth := e.PointerWidth()
	typ := instr.Type().(*types.Slice)
	elemWidth := NewConstantExpr(uint64(e.Sizeof(typ.Elem()))/8, pointerWidth)

	// Set index defaults.
	if lo == nil {
		lo = NewConstantExpr64(0)
	}
	if hi == nil {
		hi = state.selectIntAt(x, 1)
	}
	if max == nil {
		max = state.selectIntAt(x, 2)
	}

	// Data is offset based on element width and low value.
	prevData := state.selectIntAt(x, 0)
	data := newAddExpr(prevData, newMulExpr(lo, elemWidth))

	// Len is the high subtracted from the low.
	length := newSubExpr(hi, lo)

	// Capacity is max subtracted from low if 3-index slice. Otherwise use previous capacity.
	capacity := newSubExpr(max, lo)

	// Copy to new header with updated data/len/cap.
	_, hdr := state.Alloc((pointerWidth / 8) * 3)
	hdr = state.storeIntAt(hdr, 0, data)     // data
	hdr = state.storeIntAt(hdr, 1, length)   // len
	hdr = state.storeIntAt(hdr, 2, capacity) // cap
	state.heap = state.heap.Set(hdr.ID, hdr)

	// Bind header to instruction.
	state.Frame().bind(instr, hdr)

	return nil
}

func (e *Executor) executeTypeAssertInstr(state *ExecutionState, instr *ssa.TypeAssert) error {
	return fmt.Errorf("glee.Executor: type assertion is not supported")
}

func (e *Executor) executeReturnInstr(state *ExecutionState, instr *ssa.Return) error {
	// Assign return values to call instruction results.
	if frame := state.CallerFrame(); frame != nil {
		// Retrieve results from this frame.
		results := make(Tuple, len(instr.Results))
		for i := range results {
			fmt.Printf("dbg/ret %#v\n", instr.Results[i])
			results[i] = state.Eval(instr.Results[i])
		}

		// Assign value to caller
		call := frame.Instr()
		if call, ok := call.(*ssa.Call); ok {
			switch len(results) {
			case 0:
			case 1:
				frame.bind(call, results[0])
			default:
				frame.bind(call, results)
			}
		}

		// Split off new state with same constraints so we can maintain position.
		log.Print("[fork] return")
		newState := state.Fork(nil)
		newState.id = e.nextStateID()
		newState.Pop()
		e.Searcher.AddState(newState)
	}

	return nil
}

func (e *Executor) executeIfInstr(state *ExecutionState, instr *ssa.If) error {
	cond := state.Eval(instr.Cond).(Expr)
	block := instr.Block()

	// Add the false branch if it is valid.
	if satisfiable, _, err := e.Solver.Solve(append(state.constraints, NewNotExpr(cond)), nil); err != nil {
		return err
	} else if satisfiable {
		log.Print("[fork] condition false")
		newState := state.Fork(NewNotExpr(cond))
		newState.id = e.nextStateID()
		newState.Frame().jump(block.Succs[1])
		e.Searcher.AddState(newState)
	}

	// Add the true branch if it is satisfiable.
	if satisfiable, _, err := e.Solver.Solve(append(state.constraints, cond), nil); err != nil {
		return err
	} else if satisfiable {
		log.Print("[fork] condition true")
		newState := state.Fork(cond)
		newState.id = e.nextStateID()
		newState.Frame().jump(block.Succs[0])
		e.Searcher.AddState(newState)
	}

	return nil
}

func (e *Executor) executeUnOpInstr(state *ExecutionState, instr *ssa.UnOp) error {
	switch instr.Op {
	case token.NOT:
		return e.executeUnOpNotInstr(state, instr)
	case token.SUB:
		return e.executeUnOpSubInstr(state, instr)
	case token.ARROW:
		return e.executeUnOpArrowInstr(state, instr)
	case token.MUL:
		return e.executeUnOpMulInstr(state, instr)
	case token.XOR:
		return e.executeUnOpXorInstr(state, instr)
	default:
		return errors.New("invalid UnOp operator")
	}
}

func (e *Executor) executeUnOpNotInstr(state *ExecutionState, instr *ssa.UnOp) error {
	return fmt.Errorf("glee.Executor: not operator is not supported")
}

func (e *Executor) executeUnOpSubInstr(state *ExecutionState, instr *ssa.UnOp) error {
	return fmt.Errorf("glee.Executor: negation operator is not supported")
}

func (e *Executor) executeUnOpArrowInstr(state *ExecutionState, instr *ssa.UnOp) error {
	return fmt.Errorf("glee.Executor: arrow operator is not supported")
}

func (e *Executor) executeUnOpMulInstr(state *ExecutionState, instr *ssa.UnOp) error {
	width := e.Sizeof(instr.Type())

	// Find allocation by address.
	addr := state.Eval(instr.X).(*ConstantExpr)
	base, array := state.findAllocContainingAddr(addr)
	assert(array != nil, "UnOp(MUL): allocation not found: addr=%d", addr.Value)

	// Extract value from the allocation and bind it to the instruction.
	// Simple data types (such as ints) are extracted as expressions.
	// Complex data types such as interfaces are extracted as arrays.
	if isExprType(instr.Type()) {
		state.Frame().bind(instr, array.Select(newSubExpr(addr, base), width, e.IsLittleEndian()))
	} else {
		indexExpr := newSubExpr(addr, base)
		_, dst := state.Alloc(width / 8)
		for i := uint64(0); i < uint64(dst.Size); i++ {
			arrayIndex := newAddExpr(indexExpr, NewConstantExpr(i, e.PointerWidth()))
			dst.storeByte(NewConstantExpr64(i), array.selectByte(arrayIndex))
		}
		state.heap = state.heap.Set(dst.ID, dst)

		state.Frame().bind(instr, dst)
	}

	return nil
}

func (e *Executor) executeUnOpXorInstr(state *ExecutionState, instr *ssa.UnOp) error {
	return fmt.Errorf("glee.Executor: xor operator is not supported")
}

func (e *Executor) executeJumpInstr(state *ExecutionState, instr *ssa.Jump) error {
	state.Frame().jump(instr.Block().Succs[0])
	return nil
}

func (e *Executor) executePhiInstr(state *ExecutionState, instr *ssa.Phi) error {
	i := basicBlockIndex(state.Frame().block.Preds, state.Frame().prev)
	assert(i >= 0, "phi basic block not found")

	state.Frame().bind(instr, state.Eval(instr.Edges[i]))
	return nil
}

func (e *Executor) executeStoreInstr(state *ExecutionState, instr *ssa.Store) error {
	// Retrieve address from stack frame.
	addr, ok := state.EvalAsConstantExpr(instr.Addr)
	if !ok {
		return fmt.Errorf("cannot store using symbolic addresses")
	}

	// Copy value if it is an array.
	switch val := state.Eval(instr.Val).(type) {
	case *Array:
		state.Copy(addr, val)
		return nil
	case Expr:
		state.Store(addr, val)
		return nil
	default:
		return fmt.Errorf("unexpected store value: %#v", val)
	}
}

func (e *Executor) Sizes() types.Sizes {
	return types.SizesFor("gc", e.Arch)
}

func (e *Executor) Sizeof(typ types.Type) uint {
	return uint(e.Sizes().Sizeof(typ)) * 8
}

func (e *Executor) PointerWidth() uint {
	return e.Sizeof((*types.Pointer)(nil))
}

// MaxAllocSize returns the maximum allocation size.
func (e *Executor) MaxAllocSize() uint {
	if e.PointerWidth() == 32 {
		return 1 * 1024 * 1024 // 1MB
	}
	return 256 * 1024 * 1024 // 256MB
}

// IsLittleEndian returns true if the target architecture is little endian.
func (e *Executor) IsLittleEndian() bool {
	switch e.Arch {
	case "ppc64", "mips", "mips64":
		return false
	default:
		return true
	}
}

// FunctionHandler represents special execution of an SSA function call.
//
// Once registered with the Executor, all invocations of the function will be
// delegated to the FunctionHandler.
type FunctionHandler func(state *ExecutionState, instr *ssa.Call) error

// funcKey represents a key for registering a FunctionHandler with the Executor.
type funcKey struct {
	path string // package name
	name string // function name
}

// Assert adds a constraint to the current execution state.
func Assert(cond bool) {}

// execAssert represents a function handler for adding an assertion to the current state.
func execAssert(state *ExecutionState, instr *ssa.Call) error {
	_, args := state.ExtractCall(instr)

	cond, ok := args[0].(Expr)
	if !ok {
		return fmt.Errorf("glee.Assert(): unable to assert non-expression: %T", args[0])
	}

	state.AddConstraint(cond)
	return nil
}

// Byte returns a symbolic byte.
func Byte() byte { return 0 }

// Int returns a symbolic signed integer with the current execution engine's integer width.
func Int() int { return 0 }

// Int8 returns a symbolic 8-bit signed integer.
func Int8() int8 { return 0 }

// Int16 returns a symbolic 16-bit signed integer.
func Int16() int16 { return 0 }

// Int32 returns a symbolic 32-bit signed integer.
func Int32() int32 { return 0 }

// Int64 returns a symbolic 64-bit signed integer.
func Int64() int64 { return 0 }

func Uint() uint     { return 0 }
func Uint8() uint8   { return 0 }
func Uint16() uint16 { return 0 }
func Uint32() uint32 { return 0 }
func Uint64() uint64 { return 0 }

// execInt represents a function handler for all int & uint special functions.
func execInt(state *ExecutionState, instr *ssa.Call) error {
	width := state.Executor().Sizeof(instr.Type())
	_, array := state.Alloc(width / 8)
	state.Frame().bind(instr, array.Select(NewConstantExpr(0, 32), width, state.Executor().IsLittleEndian()))
	return nil
}

// String returns a symbolic string that is n bytes long.
func String(n int) string { return "" }

// execString represents a function handler for the String() function.
func execString(state *ExecutionState, instr *ssa.Call) error {
	_, args := state.ExtractCall(instr)

	n, ok := args[0].(*ConstantExpr)
	if !ok {
		return fmt.Errorf("glee.String(): only constant size allowed")
	}

	// Allocate underlying bytes.
	_, array := state.Alloc(uint(n.Value))

	// Bind array to instruction.
	state.Frame().bind(instr, array)
	return nil
}

// ByteSlice returns a symbolic byte slice that is n bytes long.
func ByteSlice(n int) []byte { return nil }

// execByteSlice represents a function handler for the ByteSlice() function.
func execByteSlice(state *ExecutionState, instr *ssa.Call) error {
	_, args := state.ExtractCall(instr)

	n, ok := args[0].(*ConstantExpr)
	if !ok {
		return fmt.Errorf("glee.ByteSlice(): only constant size allowed")
	}

	// Allocate underlying byte array.
	addr, _ := state.Alloc(uint(n.Value))

	// Allocate slice header array.
	pointerWidth := state.Executor().PointerWidth()
	_, hdr := state.Alloc((pointerWidth / 8) * 3)
	hdr = state.storeIntAt(hdr, 0, addr) // data
	hdr = state.storeIntAt(hdr, 1, n)    // len
	hdr = state.storeIntAt(hdr, 2, n)    // cap
	state.heap = state.heap.Set(hdr.ID, hdr)

	// Bind header to instruction.
	state.Frame().bind(instr, hdr)

	return nil
}

// execCopy represents a function handler for the builtin copy() function.
func execCopy(state *ExecutionState, instr *ssa.Call) error {
	_, args := state.ExtractCall(instr)

	// Retrieve underlying array, offset & size of destination.
	dstType := instr.Call.Args[1].Type().(*types.Slice)
	dstHeader := args[0].(*Array)
	dstData, ok := state.selectIntAt(dstHeader, 0).(*ConstantExpr)
	if !ok {
		return fmt.Errorf("glee: copy() expects constant dst slice data address")
	}
	dstLen, ok := state.selectIntAt(dstHeader, 1).(*ConstantExpr)
	if !ok {
		return fmt.Errorf("glee: copy() expects constant dst slice len")
	}
	dstBase, dstArray := state.findAllocContainingAddr(dstData)
	if dstArray == nil {
		return fmt.Errorf("glee: dst slice data not found: %d", dstData.Value)
	}
	dstOffset := dstData.Value - dstBase.Value
	dstSize := dstLen.Value * uint64(state.executor.Sizeof(dstType.Elem())/8)

	// Determine source raw data.
	// For a slice it's the Header.Data field. For a string it's the raw data.
	var srcArray *Array
	var srcOffset, srcSize uint64
	switch typ := instr.Call.Args[1].Type().(type) {
	case *types.Slice:
		srcHeader := args[1].(*Array)
		srcData, ok := state.selectIntAt(srcHeader, 0).(*ConstantExpr)
		if !ok {
			return fmt.Errorf("glee: copy() expects constant src slice data address")
		}
		srcLen, ok := state.selectIntAt(srcHeader, 1).(*ConstantExpr)
		if !ok {
			return fmt.Errorf("glee: copy() expects constant src slice len")
		}
		var srcBase *ConstantExpr
		srcBase, srcArray = state.findAllocContainingAddr(srcData)
		if srcArray == nil {
			return fmt.Errorf("glee: src slice data not found: %d", srcData.Value)
		}
		srcOffset = srcData.Value - srcBase.Value
		srcSize = srcLen.Value * uint64(state.executor.Sizeof(typ.Elem())/8)

	case *types.Basic:
		srcArray = args[0].(*Array)
		srcOffset, srcSize = 0, uint64(srcArray.Size)
	default:
		return fmt.Errorf("glee: invalid copy() src type: %s", typ)
	}

	// Validate that source size not larger than destination size.
	if srcSize > dstSize {
		state.status = ExecutionStatusPanicked
		state.reason = "copy out of range"
		return nil
	}

	// Copy all the bytes from src to dst.
	other := dstArray.Clone()
	for i := uint64(0); i < srcSize; i++ {
		dstIndex := NewConstantExpr64(dstOffset + i)
		srcIndex := NewConstantExpr64(srcOffset + i)
		other.storeByte(dstIndex, srcArray.selectByte(srcIndex))
	}

	// Update the heap data.
	state.heap = state.heap.Set(dstBase.Value, other)

	return nil
}

// execLen represents a function handler for the builtin len() function.
func execLen(state *ExecutionState, instr *ssa.Call) error {
	_, args := state.ExtractCall(instr)
	arg := args[0].(*Array)

	switch typ := instr.Call.Args[0].Type().(type) {
	case *types.Slice:
		v, ok := state.selectIntAt(arg, 1).(*ConstantExpr)
		if !ok {
			return fmt.Errorf("glee: len() expects constant slice len")
		}
		state.Frame().bind(instr, v)
		return nil
	case *types.Basic:
		state.Frame().bind(instr, NewConstantExpr64(uint64(arg.Size)))
		return nil
	default:
		return fmt.Errorf("glee: invalid len() arg type: %s", typ)
	}
}

// execTestingFatal represents a function handler for the testing.Fatal() function.
func execTestingFatal(state *ExecutionState, instr *ssa.Call) error {
	panic("TODO")
}

// isValidOSArch returns true if the OS & architecture combination are valid.
func isValidOSArch(os, arch string) bool {
	switch fmt.Sprintf("%s/%s", os, arch) {
	case "android/386",
		"android/amd64",
		"android/arm",
		"android/arm64",
		"darwin/386",
		"darwin/amd64",
		"darwin/arm",
		"darwin/arm64",
		"dragonfly/amd64",
		"freebsd/386",
		"freebsd/amd64",
		"freebsd/arm",
		"js/wasm",
		"linux/386",
		"linux/amd64",
		"linux/arm",
		"linux/arm64",
		"linux/mips",
		"linux/mips64",
		"linux/mips64le",
		"linux/mipsle",
		"linux/ppc64",
		"linux/ppc64le",
		"linux/riscv64",
		"linux/s390x",
		"nacl/386",
		"nacl/amd64p32",
		"nacl/arm",
		"netbsd/386",
		"netbsd/amd64",
		"netbsd/arm",
		"openbsd/386",
		"openbsd/amd64",
		"openbsd/arm",
		"plan9/386",
		"plan9/amd64",
		"plan9/arm",
		"solaris/amd64",
		"windows/386",
		"windows/amd64":
		return true
	default:
		return false
	}
}

func structFields(typ *types.Struct) []*types.Var {
	a := make([]*types.Var, typ.NumFields())
	for i := range a {
		a[i] = typ.Field(i)
	}
	return a
}

// basicBlockIndex returns the index of v within a. Returns -1 if v is not in a.
func basicBlockIndex(a []*ssa.BasicBlock, v *ssa.BasicBlock) int {
	for i := range a {
		if a[i] == v {
			return i
		}
	}
	return -1
}

// deref returns the underlying data type if typ is a pointer. Otherwise returns typ.
func deref(typ types.Type) types.Type {
	if p, ok := typ.Underlying().(*types.Pointer); ok {
		return p.Elem()
	}
	return typ
}

// isPointerType returns true if typ is a pointer type.
func isPointerType(typ types.Type) bool {
	_, ok := typ.Underlying().(*types.Pointer)
	return ok
}

// programTypes returns a sorted list of all program types.
func programTypes(prog *ssa.Program) []types.Type {
	// Collect every referenced type.
	m := make(map[types.Type]struct{})
	for _, pkg := range prog.AllPackages() {
		for _, member := range pkg.Members {
			m[member.Type()] = struct{}{}
			if fn, ok := member.(*ssa.Function); ok {
				addFunctionTypes(fn, m)
			}
		}
	}

	// Convert to a slice sorted by name.
	a := make([]types.Type, 0, len(m))
	for typ := range m {
		a = append(a, typ)
	}
	sort.Slice(a, func(i, j int) bool { return a[i].String() < a[j].String() })

	return a
}

// addFunctionTypes adds all types referred to in fn to the map.
// Recursively adds anonymous functions.
func addFunctionTypes(fn *ssa.Function, m map[types.Type]struct{}) {
	for _, param := range fn.Params {
		m[param.Type()] = struct{}{}
	}

	for _, blk := range fn.Blocks {
		for _, instr := range blk.Instrs {
			if value, ok := instr.(ssa.Value); ok {
				m[value.Type()] = struct{}{}
			}
		}
	}

	for _, anon := range fn.AnonFuncs {
		addFunctionTypes(anon, m)
	}
}

// isExprType returns true if typ is stored as an Expr.
// Only applies to boolean and integer values.
func isExprType(typ types.Type) bool {
	if typ, ok := typ.(*types.Basic); ok {
		return typ.Info()&types.IsBoolean != 0 || typ.Info()&types.IsInteger != 0
	}
	return false
}

// Solver represents a logical constraint solver.
type Solver interface {
	// Returns the satisfiability of the set of constraints. If the formula
	// is satisfiable, a valid value is returned for each array passed in.
	Solve(contraints []Expr, arrays []*Array) (satisfiable bool, values [][]byte, err error)
}

// Searcher represents a strategy for finding the next execution state to execute.
type Searcher interface {
	// Returns the next state to explore.
	SelectState() *ExecutionState

	// Adds states to the current searcher.
	AddState(state *ExecutionState)
}

var _ Searcher = (*MultiSearcher)(nil)

// MultiSearcher represents a Searcher that chooses a searcher round-robin.
type MultiSearcher struct {
	searchers []Searcher
	index     int
}

// NewMultiSearcher returns a new instance of MultiSearcher.
func NewMultiSearcher(searchers ...Searcher) *MultiSearcher {
	return &MultiSearcher{searchers: searchers}
}

// SelectState returns the next state to explore from the next searcher.
func (s *MultiSearcher) SelectState() *ExecutionState {
	searcher := s.searchers[s.index]
	if s.index++; s.index >= len(s.searchers) {
		s.index = 0
	}
	return searcher.SelectState()
}

// AddState adds a new state to the searcher.
func (s *MultiSearcher) AddState(state *ExecutionState) {
	for _, searcher := range s.searchers {
		searcher.AddState(state)
	}
}

// DFSSearcher represents a searcher with a depth-first search strategy.
type DFSSearcher struct {
	states []*ExecutionState
}

// NewDFSSearcher returns a new instance of DFSSearcher.
func NewDFSSearcher() *DFSSearcher {
	return &DFSSearcher{}
}

// SelectState returns the next execution state to explore.
func (s *DFSSearcher) SelectState() *ExecutionState {
	if len(s.states) == 0 {
		return nil
	}
	state := s.states[len(s.states)-1]
	s.states = s.states[:len(s.states)-1]
	return state
}

// AddState adds a new state to the searcher.
func (s *DFSSearcher) AddState(state *ExecutionState) {
	s.states = append(s.states, state)
}

// BFSSearcher represents a searcher with a breadth-first search strategy.
type BFSSearcher struct {
	states []*ExecutionState
}

// NewBFSSearcher returns a new instance of BFSSearcher.
func NewBFSSearcher() *BFSSearcher {
	return &BFSSearcher{}
}

// SelectState returns the next execution state to explore.
func (s *BFSSearcher) SelectState() *ExecutionState {
	if len(s.states) == 0 {
		return nil
	}
	state := s.states[0]
	s.states = s.states[1:]
	return state
}

// AddState adds a new state to the searcher.
func (s *BFSSearcher) AddState(state *ExecutionState) {
	s.states = append(s.states, state)
}

type RandomSearcher struct {
	states []*ExecutionState
	rand   *rand.Rand
}

func NewRandomSearcher(rand *rand.Rand) *RandomSearcher {
	return &RandomSearcher{
		rand: rand,
	}
}

// SelectState returns a random execution state to explore.
func (s *RandomSearcher) SelectState() *ExecutionState {
	if len(s.states) == 0 {
		return nil
	}
	i := s.rand.Intn(len(s.states))
	state := s.states[i]
	s.states = append(s.states[:i], s.states[i+1:]...)
	return state
}

// AddState adds a new state to the searcher.
func (s *RandomSearcher) AddState(state *ExecutionState) {
	s.states = append(s.states, state)
}

// RandomPathSearcher randomly selects a path from the executor's state tree.
type RandomPathSearcher struct {
	executor *Executor
	rand     *rand.Rand
}

// NewRandomPathSearcher returns a new instance of RandomPathSearcher.
func NewRandomPathSearcher(executor *Executor, rand *rand.Rand) *RandomPathSearcher {
	return &RandomPathSearcher{
		executor: executor,
		rand:     rand,
	}
}

// SelectState returns a random leaf execution state from the executor.
func (s *RandomPathSearcher) SelectState() *ExecutionState {
	state := s.executor.root
	if state == nil {
		return nil
	}

	for {
		// Return if leaf node.
		if len(state.children) == 0 {
			return state
		}

		// Otherwise randomly choose child.
		state = state.children[s.rand.Intn(len(state.children))]
	}
}

// AddState is a no-op. Searcher finds states from the executor.
func (s *RandomPathSearcher) AddState(state *ExecutionState) {}
