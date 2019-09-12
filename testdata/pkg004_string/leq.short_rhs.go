package main

import (
	"github.com/benbjohnson/glee"
)

func leqShortRHS() {
	a := glee.String(3)
	b := glee.String(2)
	glee.Assert(a[0] == b[0])
	glee.Assert(a[1] == b[1])

	if a <= b {
		return
	}
	return
}
