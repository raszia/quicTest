package main

import (
	"fmt"
	"net/http"
)

func createBigString() string {
	var s string
	for i := 0; i < 1000; i++ {
		s += "Hello, World!"
	}
	return s
}

func (h *zeroHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, createBigString())
}
