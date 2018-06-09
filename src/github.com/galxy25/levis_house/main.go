package main

import (
	"net/http"
)

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func main() {
	// Serve web files in the static directory
	http.Handle("/", http.FileServer(http.Dir("./static")))
	// Expose a health check endpoint
	http.HandleFunc("/ping", ping)
	if err := http.ListenAndServe(":8081", nil); err != nil {
		panic(err)
	}
}
