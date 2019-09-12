package main

import (
	"github.com/benbjohnson/glee"
)

func leqImpossible() {
	a := glee.String(3)
	b := glee.String(3)
	glee.Assert(a[0] == b[0])
	glee.Assert(a[1] > b[1]) // invalidate leq
	glee.Assert(a[2] < b[2])

	if a <= b {
		return
	}
	return
}
