package main

import (
	"github.com/benbjohnson/glee"
)

func simple() {
	var t T
	t.A = 5
	t.B = glee.Int()
	t.C = 7
	t.D = 8

	if int(t.A)+t.B == t.C {
		return
	}
	return
}

type T struct {
	A    int8
	B, C int
	D    int32
}
