package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err == flag.ErrHelp {
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	var cmd string
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}

	switch cmd {
	case "", "-h", "--help", "help":
		usage()
		return flag.ErrHelp
	case "generate":
		return NewGenerateCommand().Run(ctx, args)
	default:
		return fmt.Errorf(`glee %s: unknown command`, cmd)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `
Glee is a tool for symbolic execution of Go code.

Usage:

	glee <command> [arguments]

The commands are:

	generate    generate test cases
	help        this screen
`[1:])
}
