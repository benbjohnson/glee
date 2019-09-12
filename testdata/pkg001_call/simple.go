package main

import (
	"github.com/benbjohnson/glee"
)

func caller() {
	x := glee.Int8()
	y := glee.Int16()
	z := callee(x, y)
	if z == 0xAABB {
		return
	}
}

func callee(a int8, b int16) int32 {
	x := int32(a) * int32(b)
	if x > 10 {
		return x + 1
	}
	return x - 1
}
