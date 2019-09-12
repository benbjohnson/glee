package main

import (
	"github.com/benbjohnson/glee"
)

func changeInterface() {
	x := glee.Int()
	var u U = T(x)
	var v V = u

	if v.Add(10) == 100 {
		return
	}
	return
}

type T int

func (t T) Add(i int) int {
	return int(t) + i
}

func (t T) Sub(i int) int {
	return int(t) - i
}

type V interface {
	Add(i int) int
}

type U interface {
	Add(i int) int
	Sub(i int) int
}
