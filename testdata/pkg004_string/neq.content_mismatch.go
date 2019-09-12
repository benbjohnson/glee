package main

import (
	"github.com/benbjohnson/glee"
)

func neqContentMismatch() {
	a := glee.String(2)
	b := glee.String(2)

	if a != b {
		return
	}
	return
}
