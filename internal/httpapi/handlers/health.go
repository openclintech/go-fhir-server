package handlers

import (
	"net/http"
	"time"

	"go-fhir-server/internal/httpapi/respond"
)

func Ping() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			respond.JSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"}, "application/json")
			return
		}
		respond.JSON(w, http.StatusOK, map[string]any{
			"pong": true,
			"time": time.Now().UTC().Format(time.RFC3339),
		}, "application/json")
	})
}
