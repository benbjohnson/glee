package main

import (
	"github.com/benbjohnson/glee"
)

func neqLengthMismatch() {
	a := glee.String(2)
	b := glee.String(3)

	if a != b {
		return
	}
	return
}
