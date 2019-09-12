package main

import (
	"github.com/benbjohnson/glee"
)

func byteSliceIndexAddr() {
	a := glee.ByteSlice(4)
	b := make([]byte, 2, 3)
	b[0] = a[2]
	b[1] = a[1]

	if string(b) == "XY" {
		return
	}
	return
}
