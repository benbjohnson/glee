package main

import (
	"github.com/benbjohnson/glee"
)

func stringSlice() {
	a := glee.String(4)

	if a[1:3] == "XY" {
		return
	}
	return
}
