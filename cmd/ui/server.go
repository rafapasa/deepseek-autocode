package ui

import (
	_ "embed"
	"fmt"
	"net/http"
)

//go:embed index.html
var indexHTML []byte

func Start(port string) error {
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
	mux.HandleFunc("/api/config", handleConfig)
	mux.HandleFunc("/api/env", handleEnv)
	mux.HandleFunc("/api/issues", handleIssues)
	mux.HandleFunc("/api/run", handleRun)
	mux.HandleFunc("/api/stream/", handleStream)
	mux.HandleFunc("/api/stop", handleStop)

	fmt.Printf("[ds-ac ui] http://localhost:%s\n", port)
	return http.ListenAndServe(":"+port, mux)
}
