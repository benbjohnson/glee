package main

import (
	"github.com/benbjohnson/glee"
)

func sliceInterface() {
	x, y := glee.Int(), glee.Int()
	a := make([]T1, 2)
	a[0] = X1(x)
	a[1] = Y1(y)

	if a[0].Val() == a[1].Val() {
		return
	}
	return
}

type X1 int

func (x X1) Val() int {
	return int(x) + 10
}

type Y1 int

func (y Y1) Val() int {
	return int(y) + 20
}

type T1 interface {
	Val() int
}
