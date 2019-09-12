package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/format"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/benbjohnson/glee"
	"github.com/benbjohnson/glee/go/ast/astutil"
	"github.com/benbjohnson/glee/z3"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var (
	SymbolicTestPrefix = "SymbolicTest"
)

// GenerateCommand represents a command for generating test cases.
type GenerateCommand struct{}

// NewGenerateCommand returns a new instance of GenerateCommand.
func NewGenerateCommand() *GenerateCommand {
	return &GenerateCommand{}
}

// Run executes the "generate" subcommand.
func (cmd *GenerateCommand) Run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("glee-generate", flag.ContinueOnError)
	verbose := fs.Bool("v", false, "verbose")
	fs.Usage = cmd.usage
	if err := fs.Parse(args); err != nil {
		return err
	} else if fs.NArg() == 0 {
		return fmt.Errorf("package required")
	} else if fs.NArg() > 1 {
		return fmt.Errorf("too many packages specified")
	}

	log.SetFlags(0)
	if !*verbose {
		log.SetOutput(ioutil.Discard)
	}

	// Load the initial set of packages.
	initial, err := packages.Load(&packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: true,
	}, fs.Args()...)
	if err != nil {
		return err
	} else if packages.PrintErrors(initial) > 0 {
		return fmt.Errorf("packages contain errors")
	}

	// Build program in SSA form.
	prog, pkgs := ssautil.AllPackages(initial, ssa.BuilderMode(0))
	for i, pkg := range pkgs {
		if pkg == nil {
			return fmt.Errorf("cannot build SSA for package %s", initial[i])
		}
		pkg.SetDebugMode(true)
	}
	prog.Build()

	// Ensure program depends on runtime package.
	if prog.ImportedPackage("runtime") == nil {
		return fmt.Errorf("program does not depend on runtime")
	}

	// TODO: Execute existing tests to determine test coverage.

	// Find matching glee test cases.
	var fns []*ssa.Function
	for _, pkg := range pkgs {
		for _, m := range pkg.Members {
			if m, ok := m.(*ssa.Function); ok && strings.HasPrefix(m.Name(), SymbolicTestPrefix) {
				fns = append(fns, m)
			}
		}
	}
	sort.Slice(fns, func(i, j int) bool { return fns[i].Name() < fns[j].Name() })

	// Execute functions using the symbolic execution engine.
	for _, fn := range fns {
		if err := cmd.generateFunction(ctx, fn); err != nil {
			return err
		}
	}
	return nil
}

// generateFunction performs symbolic execution over a function and generates test cases.
func (cmd *GenerateCommand) generateFunction(ctx context.Context, fn *ssa.Function) error {
	var buf bytes.Buffer
	format.Node(&buf, token.NewFileSet(), fn.Syntax())

	log.Printf("[begin]")
	log.Print(buf.String())

	z3Solver := z3.NewSolver()
	defer z3Solver.Close()

	e := glee.NewExecutor(fn)
	e.Solver = z3Solver

	for {
		state, err := e.ExecuteNextState()
		if err == glee.ErrNoStateAvailable {
			break
		} else if err != nil {
			return err
		}

		// Report when a new state occurs.
		if !state.Terminated() {
			fmt.Printf("non-terminal state#%d\n", state.ID())
			fmt.Println("")
			continue
		}

		// If we reach a terminal state then generate test case from solution.
		fmt.Printf("terminal state#%d\n", state.ID())

		// Copy the AST node for the function.
		syntax := astutil.Clone(fn.Syntax())

		// TODO: Rewrite symbolic results.
		arrays, values, err := state.Values()
		for i, array := range arrays {
			value := values[i]
			fmt.Printf("%s => %x\n", array.String(), value)
		}

		// Print new test case.
		format.Node(os.Stdout, token.NewFileSet(), syntax)
	}

	log.Print("[end]")
	log.Print("")

	return nil
}

func (cmd *GenerateCommand) usage() {
	fmt.Fprintln(os.Stderr, `
usage: glee generate [arguments] [package]

Arguments:

	-v
	    Enable verbose logging.
`[1:])
}
