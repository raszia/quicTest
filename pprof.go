package main

import (
	"log"
	"net/http"
	"net/http/pprof"
)

func pprofInit() {
	// Create a new ServeMux
	mux := http.NewServeMux()

	// Register pprof handlers
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Start your server using the custom ServeMux
	log.Fatal(http.ListenAndServe(":9911", mux))
}
