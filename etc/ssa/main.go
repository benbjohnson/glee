package main

import "fmt"

var x = 1
var st = T{A: x, B: "foo"}
var sl = []int{1, 2, 3}

func main() {
	x = 2
	sl[2] = 10

	fmt.Printf("%d %d %s %v\n", st.A, x, st.B, sl)
}

type T struct {
	A int
	B string
}
