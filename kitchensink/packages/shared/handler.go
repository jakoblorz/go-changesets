package shared

import (
	"encoding/json"
	"net/http"
)

// HealthHandler returns a simple health check handler
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
		})
	}
}
