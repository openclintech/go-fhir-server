// NOTE: This is intentionally a minimal, stdlib-only FHIR server.
// Features are added incrementally for learning and clarity.

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

func main() {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		}, "application/json")
	})

	// Minimal FHIR endpoint(s)
	// POST /fhir/{resourceType}
	mux.HandleFunc("/fhir/", fhirHandler)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func fhirHandler(w http.ResponseWriter, r *http.Request) {
	// Example paths:
	//   /fhir/Patient
	//   /fhir/Observation
	// NOTE: for now, we only support create (POST) at /fhir/{resourceType}

	path := strings.TrimPrefix(r.URL.Path, "/fhir/")
	path = strings.Trim(path, "/") // normalize trailing slashes

	if path == "" {
		httpErrorJSON(w, http.StatusNotFound, "not found")
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) != 1 {
		// Later: /fhir/{resourceType}/{id}, /_history, etc.
		httpErrorJSON(w, http.StatusNotFound, "not found")
		return
	}

	resourceTypeFromPath := parts[0]

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		httpErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Decode JSON body
	defer r.Body.Close()

	var payload map[string]any
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // optional strictness; remove later if you prefer permissive
	if err := dec.Decode(&payload); err != nil {
		httpErrorJSON(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Enforce resourceType in body
	rt, ok := payload["resourceType"]
	if !ok {
		httpErrorJSON(w, http.StatusBadRequest, "missing required field: resourceType")
		return
	}
	rtStr, ok := rt.(string)
	if !ok || strings.TrimSpace(rtStr) == "" {
		httpErrorJSON(w, http.StatusBadRequest, "resourceType must be a non-empty string")
		return
	}

	// Enforce resourceType matches the path
	if rtStr != resourceTypeFromPath {
		httpErrorJSON(w, http.StatusBadRequest, "resourceType in body must match resourceType in URL path")
		return
	}

	// MVP behavior: echo payload back
	writeJSON(w, http.StatusCreated, payload, "application/fhir+json")
}

func writeJSON(w http.ResponseWriter, status int, v any, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Minimal error shape (not an OperationOutcome yet).
// We'll upgrade to OperationOutcome in a later increment.
func httpErrorJSON(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error":   message,
		"status":  status,
		"time":    time.Now().UTC().Format(time.RFC3339),
	}, "application/json")
}
