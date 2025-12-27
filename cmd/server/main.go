// NOTE: This is intentionally a minimal, stdlib-only FHIR server.
// Features are added incrementally for learning and clarity.

package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

func main() {
	s := newServer(newMemStore())

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/fhir/", s.handleFHIR)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

type server struct {
	store *memStore
}

func newServer(store *memStore) *server {
	return &server{store: store}
}

// ---- Handlers ----

func (s *server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	}, "application/json")
}

func (s *server) handleFHIR(w http.ResponseWriter, r *http.Request) {
	// Supported routes:
	//   POST /fhir/{resourceType}         (create)
	//   GET  /fhir/{resourceType}/{id}    (read)

	path := strings.TrimPrefix(r.URL.Path, "/fhir/")
	path = strings.Trim(path, "/")
	if path == "" {
		httpErrorJSON(w, http.StatusNotFound, "not found")
		return
	}

	parts := strings.Split(path, "/")

	switch len(parts) {
	case 1:
		// /fhir/{resourceType}
		resourceType := parts[0]
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			httpErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleCreate(resourceType, w, r)
		return

	case 2:
		// /fhir/{resourceType}/{id}
		resourceType := parts[0]
		id := parts[1]
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			httpErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleRead(resourceType, id, w, r)
		return

	default:
		httpErrorJSON(w, http.StatusNotFound, "not found")
		return
	}
}

// ---- FHIR operations ----

func (s *server) handleCreate(resourceTypeFromPath string, w http.ResponseWriter, r *http.Request) {
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

	// On create: ensure id + meta.versionId/meta.lastUpdated
	if err := ensureID(payload); err != nil {
		httpErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	ensureMetaCreate(payload)

	// Store it in memory for reads
	id := payload["id"].(string)
	s.store.put(resourceTypeFromPath, id, payload)

	writeJSON(w, http.StatusCreated, payload, "application/fhir+json")
}

func (s *server) handleRead(resourceTypeFromPath, id string, w http.ResponseWriter, r *http.Request) {
	if !fhirIDRe.MatchString(id) {
		httpErrorJSON(w, http.StatusBadRequest, "id is not a valid FHIR id")
		return
	}

	res, ok := s.store.get(resourceTypeFromPath, id)
	if !ok {
		httpErrorJSON(w, http.StatusNotFound, "not found")
		return
	}

	writeJSON(w, http.StatusOK, res, "application/fhir+json")
}

// ---- Helpers ----

func writeJSON(w http.ResponseWriter, status int, v any, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Minimal error shape (not an OperationOutcome yet).
// We'll upgrade to OperationOutcome in a later increment.
func httpErrorJSON(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error":  message,
		"status": status,
		"time":   time.Now().UTC().Format(time.RFC3339),
	}, "application/json")
}

// ---- FHIR-ish primitives ----

var fhirIDRe = regexp.MustCompile(`^[A-Za-z0-9\-\.]{1,64}$`)

func ensureID(resource map[string]any) error {
	// If id present, validate it. If missing, generate one.
	if v, ok := resource["id"]; ok {
		s, ok := v.(string)
		if !ok || s == "" {
			return errf("id must be a non-empty string when provided")
		}
		if !fhirIDRe.MatchString(s) {
			return errf("id is not a valid FHIR id")
		}
		return nil
	}

	resource["id"] = newFHIRID()
	return nil
}

func ensureMetaCreate(resource map[string]any) {
	meta, _ := resource["meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		resource["meta"] = meta
	}

	meta["versionId"] = "1"
	meta["lastUpdated"] = time.Now().UTC().Format(time.RFC3339)
}

func newFHIRID() string {
	// 16 random bytes -> 32 hex chars. Valid FHIR id chars and within 64 length.
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Very unlikely; fall back to timestamp-based id.
		return "id-" + time.Now().UTC().Format("20060102150405")
	}
	return hex.EncodeToString(b)
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }

func errf(msg string) error { return simpleErr(msg) }

// ---- In-memory store ----

type memStore struct {
	mu   sync.RWMutex
	data map[string]map[string]any // key = "<ResourceType>/<id>"
}

func newMemStore() *memStore {
	return &memStore{data: make(map[string]map[string]any)}
}

func (s *memStore) put(resourceType, id string, resource map[string]any) {
	key := resourceType + "/" + id

	s.mu.Lock()
	defer s.mu.Unlock()

	// store a copy so callers can't mutate what's stored
	s.data[key] = deepCopyMap(resource)
}

func (s *memStore) get(resourceType, id string) (map[string]any, bool) {
	key := resourceType + "/" + id

	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.data[key]
	if !ok {
		return nil, false
	}
	return deepCopyMap(v), true
}

func deepCopyMap(m map[string]any) map[string]any {
	// Simple JSON round-trip copy (fine for learning; optimize later)
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}
