package main

import (
	"github.com/benbjohnson/glee"
)

func simple() {
	x := glee.Int()
	if x == 0xAABB {
		x++
	}
}
