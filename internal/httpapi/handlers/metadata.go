package handlers

import (
	"net/http"
	"time"

	"go-fhir-server/internal/httpapi/respond"
)

// Metadata returns a minimal CapabilityStatement at GET /fhir/metadata.
func Metadata() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			respond.OperationOutcome(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		// Minimal, honest CapabilityStatement for this MVP.
		cs := map[string]any{
			"resourceType": "CapabilityStatement",
			"status":       "active",
			"date":         time.Now().UTC().Format(time.RFC3339),
			"kind":         "instance",
			"fhirVersion":  "4.0.1",
			"format":       []string{"json"},
			"rest": []any{
				map[string]any{
					"mode": "server",
					"resource": []any{
						map[string]any{
							"type": "Patient",
							"interaction": []any{
								map[string]any{"code": "create"},
								map[string]any{"code": "read"},
								map[string]any{"code": "update"},
								map[string]any{"code": "delete"},
								map[string]any{"code": "search-type"},
							},
						},
					},
				},
			},
		}

		respond.JSON(w, http.StatusOK, cs, "application/fhir+json")
	})
}
