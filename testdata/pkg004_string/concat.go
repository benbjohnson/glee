package main

import (
	"github.com/benbjohnson/glee"
)

func stringConcat() {
	a := glee.String(2)
	b := glee.String(3)

	if a+"obar"+b == "foobarbaz" {
		return
	}
	return
}
