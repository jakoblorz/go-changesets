package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jakoblorz/go-changeset/kitchensink/packages/shared"
)

func main() {
	mux := http.NewServeMux()

	// Mount shared health handler
	mux.Handle("/health", shared.HealthHandler())

	// WWW-specific endpoints
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to WWW service\n")
	})

	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "About page\n")
	})

	port := ":8081"
	log.Printf("WWW service starting on %s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal(err)
	}
}
