package main

import (
	"github.com/benbjohnson/glee"
)

func convertByteSlice() {
	a := glee.String(2)
	b := []byte(a)

	if string(b) == "XY" {
		return
	}
	return
}
