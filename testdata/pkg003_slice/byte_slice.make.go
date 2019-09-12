package main

import (
	"github.com/benbjohnson/glee"
)

func byteSliceMake() {
	i, j := 2, 3
	b := make([]byte, i, j)
	b[0] = glee.Byte()
	b[1] = glee.Byte()

	if string(b) == "XY" {
		return
	}
	return
}
