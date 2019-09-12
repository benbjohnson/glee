package main

import (
	"github.com/benbjohnson/glee"
)

func sliceByteSlice() {
	a := glee.ByteSlice(4)
	b := a[1:3]
	s := string(b)

	if s == "XY" {
		return
	}
	return
}
