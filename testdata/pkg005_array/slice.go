package main

import (
	"github.com/benbjohnson/glee"
)

func arraySlice() {
	a := glee.ByteSlice(4)
	var b [4]byte
	copy(b[:], a)

	if string(b[1:3]) == "XY" {
		return
	}
	return
}
