package main

import (
	"log"
	"net/http"

	"github.com/theothertomelliott/git-backed-rest/backends/memory"
)

func main() {
	server := &Server{
		backend: memory.NewBackend(),
	}
	http.HandleFunc("/", server.handleRequest)

	port := ":8080"
	log.Printf("Starting server on %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
