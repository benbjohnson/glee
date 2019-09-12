package main

import (
	"github.com/benbjohnson/glee"
)

func stringSliceOOB() {
	a := glee.String(4)

	if a[0:8] == "XYXYXYXY" {
		return
	}
	return
}
