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

	// Auditor-specific endpoint
	mux.HandleFunc("/audit", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Auditor service running\n")
	})

	port := ":8080"
	log.Printf("Auditor service starting on %s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatal(err)
	}
}
